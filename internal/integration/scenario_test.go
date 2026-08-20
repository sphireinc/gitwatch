package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/stash"
	"github.com/jusanchez/gitwatch/internal/worktrees"
)

func TestRepositoryWorkbenchScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	if _, err := runner.Run(ctx, "init", "--", root); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "config", "user.name", "gitwatch-test"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "config", "user.email", "gitwatch-test@example.com"); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(file, []byte("initial\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("tracked.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("initial\n")}); err != nil {
		t.Fatal(err)
	}
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := git.Snapshot(ctx, discovery, 1)
	if err != nil || len(snapshot.Entries) != 1 || !snapshot.Entries[0].Unstaged {
		t.Fatalf("unexpected changed snapshot: %#v, %v", snapshot, err)
	}
	if _, err := stash.Create(ctx, runner, "scenario"); err != nil {
		t.Fatal(err)
	}
	stashes, err := stash.List(ctx, runner)
	if err != nil || len(stashes) == 0 {
		t.Fatalf("stash was not created: %#v, %v", stashes, err)
	}
	if _, err := branches.Create(ctx, runner, "scenario-branch"); err != nil {
		t.Fatal(err)
	}
	worktreePath := filepath.Join(t.TempDir(), "linked")
	if _, err := worktrees.Add(ctx, runner, worktreePath, "scenario-worktree"); err != nil {
		t.Fatal(err)
	}
	entries, err := worktrees.List(ctx, runner)
	if err != nil || len(entries) < 2 {
		t.Fatalf("worktree discovery failed: %#v, %v", entries, err)
	}
	canonicalWorktreePath, _ := filepath.EvalSymlinks(worktreePath)
	if worktrees.Occupancy(entries)["scenario-worktree"] != canonicalWorktreePath {
		t.Fatalf("worktree occupancy missing: %#v", worktrees.Occupancy(entries))
	}
}
