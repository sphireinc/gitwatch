package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/history"
	"github.com/jusanchez/gitwatch/internal/remotes"
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
	if _, err := worktrees.Remove(ctx, runner, worktreePath, false); err != nil {
		t.Fatalf("worktree remove failed: %v", err)
	}
	entries, err = worktrees.List(ctx, runner)
	if err != nil || len(entries) != 1 {
		t.Fatalf("worktree discovery after remove = %#v, err=%v", entries, err)
	}
	if _, err := worktrees.Prune(ctx, runner, true); err != nil {
		t.Fatalf("worktree prune dry run failed: %v", err)
	}
}

func TestHistoryActionsRealRepositoryScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{{"init", "--", root}, {"config", "user.name", "history-test"}, {"config", "user.email", "history@example.com"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "history.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("history.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("one\n")}); err != nil {
		t.Fatal(err)
	}
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	mainBranch := discovery.Root
	snapshot, err := git.Snapshot(ctx, discovery, 0)
	if err != nil {
		t.Fatal(err)
	}
	mainBranch = snapshot.Branch.Name
	firstSHA := snapshot.Branch.OID
	if err := os.WriteFile(path, []byte("two\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("history.txt")); err != nil {
		t.Fatal(err)
	}
	second, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("two\n")})
	if err != nil {
		t.Fatal(err)
	}
	secondSHA := second.SHA
	if _, err := runner.Run(ctx, "tag", "v-history"); err != nil {
		t.Fatal(err)
	}
	tags, err := history.ListTags(ctx, runner)
	if err != nil || len(tags) != 1 || tags[0].Name != "v-history" {
		t.Fatalf("tags = %#v, err=%v", tags, err)
	}
	if _, err := history.CreateBranchAt(ctx, runner, "from-first", firstSHA); err != nil {
		t.Fatal(err)
	}
	branchSHA, err := runner.Run(ctx, "rev-parse", "from-first")
	if err != nil || strings.TrimSpace(string(branchSHA.Stdout)) != firstSHA {
		t.Fatalf("branch target = %q, err=%v", branchSHA.Stdout, err)
	}
	if _, err := history.CheckoutCommit(ctx, runner, firstSHA); err != nil {
		t.Fatal(err)
	}
	head, err := runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(string(head.Stdout)) != firstSHA {
		t.Fatalf("detached head = %q, err=%v", head.Stdout, err)
	}
	if _, err := branches.Checkout(ctx, runner, mainBranch); err != nil {
		t.Fatal(err)
	}
	if _, err := history.Revert(ctx, runner, history.RevertConfirmation{SHA: secondSHA}, secondSHA); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "one\n" {
		t.Fatalf("reverted file = %q, err=%v", contents, err)
	}
}

func TestStashMutationsRealRepositoryScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{{"init", "--", root}, {"config", "user.name", "stash-test"}, {"config", "user.email", "stash@example.com"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "stash.txt")
	if err := os.WriteFile(path, []byte("base\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("stash.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("base\n")}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("applied\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := stash.Create(ctx, runner, "apply-test"); err != nil {
		t.Fatal(err)
	}
	entries, err := stash.List(ctx, runner)
	if err != nil || len(entries) != 1 {
		t.Fatalf("stash list after create = %#v, err=%v", entries, err)
	}
	if err := os.WriteFile(path, []byte("dirty\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := stash.ApplyChecked(ctx, runner, entries[0].Ref); err == nil {
		t.Fatal("checked stash apply accepted a dirty worktree")
	}
	if _, err := runner.Run(ctx, "restore", "--worktree", "--", "stash.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := stash.ApplyChecked(ctx, runner, entries[0].Ref); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "applied\n" {
		t.Fatalf("applied contents = %q, err=%v", contents, err)
	}
	if _, err := runner.Run(ctx, "restore", "--worktree", "--", "stash.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := stash.Drop(ctx, runner, entries[0].Ref); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("popped\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := stash.Create(ctx, runner, "pop-test"); err != nil {
		t.Fatal(err)
	}
	entries, err = stash.List(ctx, runner)
	if err != nil || len(entries) != 1 {
		t.Fatalf("stash list before pop = %#v, err=%v", entries, err)
	}
	if _, err := stash.PopChecked(ctx, runner, entries[0].Ref); err != nil {
		t.Fatal(err)
	}
	contents, err = os.ReadFile(path)
	if err != nil || string(contents) != "popped\n" {
		t.Fatalf("popped contents = %q, err=%v", contents, err)
	}
	entries, err = stash.List(ctx, runner)
	if err != nil || len(entries) != 0 {
		t.Fatalf("stash list after pop = %#v, err=%v", entries, err)
	}
	if err := os.WriteFile(path, []byte("branch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := stash.Create(ctx, runner, "branch-test"); err != nil {
		t.Fatal(err)
	}
	entries, err = stash.List(ctx, runner)
	if err != nil || len(entries) != 1 {
		t.Fatalf("stash list before branch = %#v, err=%v", entries, err)
	}
	if _, err := stash.BranchChecked(ctx, runner, "from-stash", entries[0].Ref); err != nil {
		t.Fatal(err)
	}
	branchSnapshot, err := git.Snapshot(ctx, discovery, 0)
	if err != nil || branchSnapshot.Branch.Name != "from-stash" {
		t.Fatalf("stash branch snapshot = %#v, err=%v", branchSnapshot.Branch, err)
	}
	contents, err = os.ReadFile(path)
	if err != nil || string(contents) != "branch\n" {
		t.Fatalf("branch stash contents = %q, err=%v", contents, err)
	}
}

func TestRemotePushPreviewRealRepositoryScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	bare := filepath.Join(t.TempDir(), "origin.git")
	if _, err := git.NewRunner(root).Run(ctx, "init", "-b", "main", "--", root); err != nil {
		t.Fatal(err)
	}
	runner := git.NewRunner(root)
	for _, args := range [][]string{{"config", "user.name", "remote-test"}, {"config", "user.email", "remote@example.com"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(bare, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := git.NewRunner(bare).Run(ctx, "init", "--bare"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "remote", "add", "origin", bare); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "remote.txt")
	if err := os.WriteFile(path, []byte("remote\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("remote.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("remote\n")}); err != nil {
		t.Fatal(err)
	}
	preview, err := remotes.PreviewPush(ctx, runner, "origin", "main")
	if err != nil || preview.LocalSHA == "" || preview.RemoteSHA != "" {
		t.Fatalf("initial push preview = %#v, %v", preview, err)
	}
	if _, err := remotes.Push(ctx, runner, "origin", "main", false); err != nil {
		t.Fatal(err)
	}
	preview, err = remotes.PreviewPush(ctx, runner, "origin", "main")
	if err != nil || preview.RemoteSHA != preview.LocalSHA {
		t.Fatalf("post-push preview = %#v, %v", preview, err)
	}
	if _, err := remotes.Fetch(ctx, runner, "origin"); err != nil {
		t.Fatal(err)
	}
}
