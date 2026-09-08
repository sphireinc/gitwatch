// Package customcmd defines safe, shell-free user command templates.
package customcmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

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
	if strings.TrimSpace(i.Executable) == "" {
		return nil, fmt.Errorf("custom command executable is required")
	}
	return exec.Command(i.Executable, i.Args...), nil
}

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
