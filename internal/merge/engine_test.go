package merge

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestExecuteFastForwardAndRejectsDirtyWorktree(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "base"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("base")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("base\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("feature")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("feature\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	engine := Engine{Runner: runner, Repository: dir}
	outcome := engine.Execute(ctx, Request{Repository: dir, Source: "feature", Strategy: FastForwardOnly})
	if outcome.Err != nil || outcome.Paused {
		t.Fatalf("fast-forward outcome = %#v", outcome)
	}
	if err := os.WriteFile(filepath.Join(dir, "dirty"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirty := engine.Execute(ctx, Request{Repository: dir, Source: "feature", Strategy: Regular})
	if dirty.Err == nil || !strings.Contains(dirty.Err.Error(), "clean worktree") {
		t.Fatalf("dirty outcome = %#v", dirty)
	}
}

func TestInvalidMergeSourceAndStrategyAreRejected(t *testing.T) {
	engine := Engine{Repository: "repo"}
	if got := engine.Execute(context.Background(), Request{Repository: "repo", Source: "-bad"}); !strings.Contains(got.Err.Error(), "invalid") {
		t.Fatalf("invalid source outcome = %#v", got)
	}
	if got := engine.Execute(context.Background(), Request{Repository: "repo", Source: "branch", Strategy: Strategy(99)}); got.Err == nil {
		t.Fatal("invalid strategy was not rejected")
	}
}

func TestConflictingMergeReturnsPausedStateAndAbortRestoresWorktree(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := setupMergeRepository(t, dir)
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	writeMergeFile(t, runner, dir, "feature\n", "feature")
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	writeMergeFile(t, runner, dir, "main\n", "main")
	engine := Engine{Runner: runner, Repository: dir}
	outcome := engine.Execute(ctx, Request{Repository: dir, Source: "feature", Strategy: Regular})
	if !outcome.Paused || outcome.State == nil || outcome.State.Kind().String() != "merge" {
		t.Fatalf("conflict outcome = %#v", outcome)
	}
	aborted := engine.Abort(ctx)
	if aborted.Err != nil || aborted.Paused {
		t.Fatalf("abort outcome = %#v", aborted)
	}
	status, err := runner.Run(ctx, "status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(status.Stdout)) != "" {
		t.Fatalf("status after abort = %q", status.Stdout)
	}
}

func setupMergeRepository(t *testing.T, dir string) git.Runner {
	t.Helper()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "shared"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(context.Background(), []byte("shared")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(context.Background(), git.CommitOptions{Message: []byte("base\n")}); err != nil {
		t.Fatal(err)
	}
	return runner
}

func writeMergeFile(t *testing.T, runner git.Runner, dir, contents, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "shared"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(context.Background(), []byte("shared")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(context.Background(), git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
		t.Fatal(err)
	}
}
