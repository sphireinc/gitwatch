// Package customcmd defines safe, shell-free user command templates.
package customcmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

var ErrOutputLimit = fmt.Errorf("custom command output exceeds configured limit")

// Output contains bounded command output for status or journal presentation.
type Output struct {
	Stdout, Stderr []byte
}

// Context supplies repository values to one custom command invocation.
type Context struct {
	RepositoryRoot string
	SelectedPath   string
	SelectedSHA    string
	Branch         string
	Remote         string
	Tag            string
	ProviderURL    string
}

// Definition is the JSON/configuration representation of a custom command.
// Args are argv tokens, never a shell command string.
type Definition struct {
	Name       string        `json:"name"`
	Contexts   []string      `json:"contexts,omitempty"`
	Binding    string        `json:"binding,omitempty"`
	Label      string        `json:"label,omitempty"`
	Executable string        `json:"executable"`
	Args       []string      `json:"args"`
	Directory  string        `json:"directory,omitempty"`
	Timeout    time.Duration `json:"timeout"`
	Confirm    bool          `json:"confirm"`
	Mutates    bool          `json:"mutates_repository"`
	Refresh    bool          `json:"refresh"`
}

// Invocation is a validated, expanded process request.
type Invocation struct {
	Name       string
	Label      string
	Executable string
	Args       []string
	Directory  string
	Timeout    time.Duration
	Confirm    bool
	Mutates    bool
	Refresh    bool
}

// Command creates an exec.Cmd without involving a shell. It is intentionally
// the last boundary before the operation/process owner starts the command.
func (i Invocation) Command() (*exec.Cmd, error) {
	return i.command(context.Background())
}

// CommandContext creates an argv-only process bound to ctx.
func (i Invocation) CommandContext(ctx context.Context) (*exec.Cmd, error) {
	return i.command(ctx)
}

func (i Invocation) command(ctx context.Context) (*exec.Cmd, error) {
	if strings.TrimSpace(i.Executable) == "" {
		return nil, fmt.Errorf("custom command executable is required")
	}
	command := exec.CommandContext(ctx, i.Executable, i.Args...)
	command.Dir = i.Directory
	return command, nil
}

// Run executes one validated invocation with bounded stdout/stderr. The
// context owns cancellation and timeout; no shell is ever involved.
func Run(ctx context.Context, invocation Invocation, maxBytes int) (Output, error) {
	if maxBytes <= 0 {
		return Output{}, ErrOutputLimit
	}
	command, err := invocation.CommandContext(ctx)
	if err != nil {
		return Output{}, err
	}
	stdout, stderr := &limitedBuffer{limit: maxBytes}, &limitedBuffer{limit: maxBytes}
	command.Stdout, command.Stderr = stdout, stderr
	err = command.Run()
	output := Output{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if stdout.exceeded || stderr.exceeded {
		return output, ErrOutputLimit
	}
	if err != nil {
		if message := strings.TrimSpace(string(output.Stderr)); message != "" {
			return output, fmt.Errorf("custom command %q: %w: %s", invocation.Name, err, message)
		}
		return output, fmt.Errorf("custom command %q: %w", invocation.Name, err)
	}
	return output, nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(data []byte) (int, error) {
	if b.Len()+len(data) > b.limit {
		remaining := b.limit - b.Len()
		if remaining > 0 {
			_, _ = b.Buffer.Write(data[:remaining])
		}
		b.exceeded = true
		return len(data), nil
	}
	return b.Buffer.Write(data)
}

var _ io.Writer = (*limitedBuffer)(nil)

// Expand validates and expands one command against a concrete context.
func (d Definition) Expand(ctx Context) (Invocation, error) {
	if err := d.Validate(); err != nil {
		return Invocation{}, err
	}
	values := map[string]string{
		"repo":   ctx.RepositoryRoot,
		"path":   ctx.SelectedPath,
		"sha":    ctx.SelectedSHA,
		"branch": ctx.Branch,
		"remote": ctx.Remote,
		"tag":    ctx.Tag,
		"url":    ctx.ProviderURL,
	}
	args := make([]string, len(d.Args))
	for index, token := range d.Args {
		for name, value := range values {
			placeholder := "{" + name + "}"
			if strings.Contains(token, placeholder) {
				if value == "" {
					return Invocation{}, fmt.Errorf("custom command %q requires context value %s", d.Name, placeholder)
				}
				token = strings.ReplaceAll(token, placeholder, value)
			}
		}
		if strings.ContainsAny(token, "\x00\r\n") {
			return Invocation{}, fmt.Errorf("custom command %q contains an invalid argv value", d.Name)
		}
		args[index] = token
	}
	directory := d.Directory
	if directory == "" {
		directory = ctx.RepositoryRoot
	} else if strings.Contains(directory, "{repo}") {
		if ctx.RepositoryRoot == "" {
			return Invocation{}, fmt.Errorf("custom command %q requires context value {repo}", d.Name)
		}
		directory = strings.ReplaceAll(directory, "{repo}", ctx.RepositoryRoot)
	}
	return Invocation{Name: d.Name, Label: d.displayLabel(), Executable: d.Executable, Args: args, Directory: directory, Timeout: d.Timeout, Confirm: d.Confirm, Mutates: d.Mutates, Refresh: d.Refresh || d.Mutates}, nil
}

// Validate rejects shell-shaped configuration and unknown placeholders.
func (d Definition) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("custom command name is required")
	}
	if strings.TrimSpace(d.Executable) == "" {
		return fmt.Errorf("custom command %q executable is required", d.Name)
	}
	if d.Timeout < 0 {
		return fmt.Errorf("custom command %q timeout cannot be negative", d.Name)
	}
	for _, contextName := range d.Contexts {
		switch contextName {
		case "status", "history", "compare", "github", "any":
		default:
			return fmt.Errorf("custom command %q has unknown context %q", d.Name, contextName)
		}
	}
	allowed := map[string]bool{"repo": true, "path": true, "sha": true, "branch": true, "remote": true, "tag": true, "url": true}
	for _, token := range append(append([]string(nil), d.Args...), d.Directory) {
		for start := strings.IndexByte(token, '{'); start >= 0; {
			end := strings.IndexByte(token[start:], '}')
			if end < 0 {
				return fmt.Errorf("custom command %q has an unterminated placeholder", d.Name)
			}
			name := token[start+1 : start+end]
			if !allowed[name] {
				return fmt.Errorf("custom command %q has unknown placeholder {%s}", d.Name, name)
			}
			token = token[start+end+1:]
			start = strings.IndexByte(token, '{')
		}
	}
	return nil
}

func (d Definition) displayLabel() string {
	if d.Label != "" {
		return d.Label
	}
	return d.Name
}
