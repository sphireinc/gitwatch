package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

var (
	ErrGitMissing     = errors.New("git executable not found")
	ErrNotRepository  = errors.New("not a git repository")
	ErrCommandFailed  = errors.New("git command failed")
	ErrCancelled      = errors.New("git command cancelled")
	ErrUnsupportedGit = errors.New("unsupported git version")
)

type Result struct {
	Args     []string
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
}

type CommandError struct {
	Kind   error
	Args   []string
	Result Result
	Cause  error
}

func (e *CommandError) Error() string {
	if len(e.Result.Stderr) > 0 {
		return fmt.Sprintf("%v: git %s: %s", e.Kind, strings.Join(e.Args, " "), strings.TrimSpace(string(e.Result.Stderr)))
	}
	return fmt.Sprintf("%v: git %s", e.Kind, strings.Join(e.Args, " "))
}

func (e *CommandError) Unwrap() error { return e.Kind }

type Runner struct {
	Binary string
	Dir    string
}

func NewRunner(dir string) Runner { return Runner{Binary: "git", Dir: dir} }

func (r Runner) Run(ctx context.Context, args ...string) (Result, error) {
	return r.run(ctx, nil, args...)
}

func (r Runner) RunInput(ctx context.Context, input []byte, args ...string) (Result, error) {
	return r.run(ctx, bytes.NewReader(input), args...)
}

func (r Runner) run(ctx context.Context, input io.Reader, args ...string) (Result, error) {
	binary := r.Binary
	if binary == "" {
		binary = "git"
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = r.Dir
	cmd.Stdin = input
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Args: append([]string(nil), args...), Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Duration: time.Since(start), ExitCode: 0}
	if err == nil {
		return result, nil
	}
	if ctx.Err() != nil {
		return result, &CommandError{Kind: ErrCancelled, Args: result.Args, Result: result, Cause: ctx.Err()}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.ExitCode = exitErr.ExitCode()
		kind := ErrCommandFailed
		if strings.Contains(string(result.Stderr), "not a git repository") {
			kind = ErrNotRepository
		}
		return result, &CommandError{Kind: kind, Args: result.Args, Result: result, Cause: err}
	}
	if errors.Is(err, exec.ErrNotFound) {
		return result, &CommandError{Kind: ErrGitMissing, Args: result.Args, Result: result, Cause: err}
	}
	return result, &CommandError{Kind: ErrCommandFailed, Args: result.Args, Result: result, Cause: err}
}
