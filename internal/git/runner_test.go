package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func TestRunnerClassifiesMissingGit(t *testing.T) {
	_, err := (Runner{Binary: filepath.Join(t.TempDir(), "does-not-exist"), Dir: t.TempDir()}).Run(context.Background(), "--version")
	if !errors.Is(err, ErrGitMissing) {
		t.Fatalf("expected missing-git error, got %v", err)
	}
}

func TestRunnerCancellation(t *testing.T) {
	runner := NewRunner(t.TempDir())
	if _, err := runner.Run(context.Background(), "init", "--quiet"); err != nil {
		t.Fatal(err)
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = runner.run(ctx, reader, "cat-file", "--batch")
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
