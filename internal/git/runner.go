package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	// ErrGitMissing indicates that the Git executable could not be started.
	ErrGitMissing     = errors.New("git executable not found")
	ErrNotRepository  = errors.New("not a git repository")
	ErrCommandFailed  = errors.New("git command failed")
	ErrCancelled      = errors.New("git command cancelled")
	ErrUnsupportedGit = errors.New("unsupported git version")
	ErrOutputLimit    = errors.New("git output exceeds configured limit")
)

// Result captures Git's arguments, output, exit code, and execution duration.
type Result struct {
	Args     []string
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
}

// OutputStream identifies the process stream associated with a streamed
// output chunk.
type OutputStream string

const (
	StdoutStream OutputStream = "stdout"
	StderrStream OutputStream = "stderr"
)

// OutputChunk is a bounded, process-output fragment. Data is copied before
// delivery and is safe for the callback to retain.
type OutputChunk struct {
	Stream OutputStream
	Data   []byte
}

// CommandError classifies a failed Git invocation and retains its result.
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

// Runner executes Git with an argument vector in a specific directory.
type Runner struct {
	Binary string
	Dir    string
	Env    []string
}

// NewRunner creates a Runner that invokes Git from dir.
func NewRunner(dir string) Runner { return Runner{Binary: "git", Dir: dir} }

// Run executes Git without stdin and returns its captured result.
func (r Runner) Run(ctx context.Context, args ...string) (Result, error) {
	return r.run(ctx, nil, 0, args...)
}

// RunInput executes Git with input connected to stdin.
func (r Runner) RunInput(ctx context.Context, input []byte, args ...string) (Result, error) {
	return r.run(ctx, bytes.NewReader(input), 0, args...)
}

// RunBounded executes Git while retaining at most maxBytes of stdout. It is
// intended for object/content reads where Git may legitimately produce much
// more data than the UI should hold in memory.
func (r Runner) RunBounded(ctx context.Context, maxBytes int, args ...string) (Result, error) {
	if maxBytes <= 0 {
		return Result{Args: append([]string(nil), args...)}, fmt.Errorf("git output limit must be positive: %w", ErrOutputLimit)
	}
	return r.run(ctx, nil, maxBytes, args...)
}

// RunBoundedStreaming executes Git with argv-only semantics and delivers
// bounded stdout/stderr fragments while the process is running. The callback
// may be invoked concurrently for the two streams and must not block on
// process shutdown. Output is retained up to maxBytes per stream for the
// returned Result.
func (r Runner) RunBoundedStreaming(ctx context.Context, maxBytes int, onChunk func(OutputChunk), args ...string) (Result, error) {
	if maxBytes <= 0 {
		return Result{Args: append([]string(nil), args...)}, fmt.Errorf("git output limit must be positive: %w", ErrOutputLimit)
	}
	return r.runWithCallback(ctx, nil, maxBytes, onChunk, args...)
}

type boundedBuffer struct {
	data  []byte
	limit int
	over  bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	remaining := b.limit - len(b.data)
	if len(p) > remaining {
		if remaining > 0 {
			b.data = append(b.data, p[:remaining]...)
		}
		b.over = true
	} else {
		b.data = append(b.data, p...)
	}
	return len(p), nil
}

func (r Runner) run(ctx context.Context, input io.Reader, maxBytes int, args ...string) (Result, error) {
	return r.runWithCallback(ctx, input, maxBytes, nil, args...)
}

type streamingBuffer struct {
	mu      sync.Mutex
	data    []byte
	limit   int
	stream  OutputStream
	over    bool
	onChunk func(OutputChunk)
}

func (b *streamingBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	remaining := b.limit - len(b.data)
	accepted := p
	if len(p) > remaining {
		if remaining > 0 {
			accepted = p[:remaining]
		} else {
			accepted = nil
		}
		b.over = true
	}
	if len(accepted) > 0 {
		b.data = append(b.data, accepted...)
	}
	chunk := append([]byte(nil), accepted...)
	callback := b.onChunk
	stream := b.stream
	b.mu.Unlock()
	if len(chunk) > 0 && callback != nil {
		callback(OutputChunk{Stream: stream, Data: chunk})
	}
	return len(p), nil
}

func (b *streamingBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.data...)
}

func (b *streamingBuffer) Exceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.over
}

func (r Runner) runWithCallback(ctx context.Context, input io.Reader, maxBytes int, onChunk func(OutputChunk), args ...string) (Result, error) {
	binary := r.Binary
	if binary == "" {
		binary = "git"
	}
	start := time.Now()
	cmd := exec.CommandContext(ctx, binary, args...)
	configureCommandCancellation(cmd)
	cmd.Dir = r.Dir
	if len(r.Env) > 0 {
		cmd.Env = append(os.Environ(), r.Env...)
	}
	cmd.Stdin = input
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var bounded *boundedBuffer
	var streamedOut, streamedErr *streamingBuffer
	if onChunk != nil {
		streamedOut = &streamingBuffer{limit: maxBytes, stream: StdoutStream, onChunk: onChunk}
		streamedErr = &streamingBuffer{limit: maxBytes, stream: StderrStream, onChunk: onChunk}
		cmd.Stdout, cmd.Stderr = streamedOut, streamedErr
	} else if maxBytes > 0 {
		bounded = &boundedBuffer{limit: maxBytes}
		cmd.Stdout = bounded
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
	}
	err := cmd.Run()
	if streamedOut != nil {
		stdout.Write(streamedOut.Bytes())
		stderr.Write(streamedErr.Bytes())
	} else if bounded != nil {
		stdout.Write(bounded.data)
	}
	result := Result{Args: append([]string(nil), args...), Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Duration: time.Since(start), ExitCode: 0}
	if (streamedOut != nil && (streamedOut.Exceeded() || streamedErr.Exceeded())) || (bounded != nil && bounded.over) {
		return result, &CommandError{Kind: ErrOutputLimit, Args: result.Args, Result: result}
	}
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
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return result, &CommandError{Kind: ErrGitMissing, Args: result.Args, Result: result, Cause: err}
	}
	return result, &CommandError{Kind: ErrCommandFailed, Args: result.Args, Result: result, Cause: err}
}
