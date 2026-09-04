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
