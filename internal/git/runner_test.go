package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRunnerUsesArgumentVector(t *testing.T) {
	dir := t.TempDir()
	path := "$(touch " + filepath.Join(dir, "pwned") + ")"
	runner := Runner{Binary: "git", Dir: dir}
	_, _ = runner.Run(context.Background(), "--version", path)
	if _, statErr := os.Stat(filepath.Join(dir, "pwned")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("argument was interpreted by a shell: %v", statErr)
	}
}

func TestRunnerCapturesFailure(t *testing.T) {
	_, err := NewRunner(t.TempDir()).Run(context.Background(), "status")
	if !errors.Is(err, ErrNotRepository) {
		t.Fatalf("expected not-repository error, got %v", err)
	}
	var commandErr *CommandError
	if !errors.As(err, &commandErr) || commandErr.Result.ExitCode == 0 || len(commandErr.Result.Stderr) == 0 {
		t.Fatalf("missing structured failure result: %#v", err)
	}
}

func TestRunnerCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable long-running Git cancellation fixture is Unix-specific")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := NewRunner(t.TempDir()).Run(ctx, "-c", "alias.version=!sleep 10", "version")
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
