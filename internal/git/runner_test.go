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
	marker := filepath.Join(t.TempDir(), "helper-started")
	runner := Runner{
		Binary: os.Args[0],
		Dir:    t.TempDir(),
		Env: []string{
			"GITWATCH_RUNNER_HELPER=1",
			"GITWATCH_RUNNER_HELPER_MARKER=" + marker,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := runner.Run(ctx, "-test.run=^TestRunnerCancellationHelper$")
		errCh <- err
	}()

	deadline := time.NewTimer(2 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer deadline.Stop()
	defer ticker.Stop()
	started := false
	for !started {
		select {
		case <-ticker.C:
			_, err := os.Stat(marker)
			started = err == nil
		case <-deadline.C:
			cancel()
			<-errCh
			t.Fatal("cancellation helper did not start")
		}
	}
	cancel()
	err := <-errCh
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestRunnerCancellationHelper(t *testing.T) {
	if os.Getenv("GITWATCH_RUNNER_HELPER") != "1" {
		return
	}
	marker := os.Getenv("GITWATCH_RUNNER_HELPER_MARKER")
	if marker == "" {
		os.Exit(2)
	}
	if err := os.WriteFile(marker, []byte("started"), 0o600); err != nil {
		os.Exit(2)
	}
	for {
		time.Sleep(time.Hour)
	}
}
