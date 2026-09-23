package redo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/undo"
)

const (
	oldHead = "0123456789012345678901234567890123456789"
	newHead = "abcdefabcdefabcdefabcdefabcdefabcdefabcd"
)

func TestPlanAllowsOnlyMatchingUndoReplay(t *testing.T) {
	snapshot := repo.Snapshot{Root: "/repo", Branch: repo.Branch{Name: "main", OID: newHead}}
	request := Request{Repository: "/repo", Kind: "undo commit", Ref: "main", OldHead: oldHead, NewHead: newHead, PostSnapshotHash: snapshotFingerprint(snapshot)}
	args, err := Plan(request, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"reset", "--soft", oldHead}
	if len(args) != len(want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args = %#v, want %#v", args, want)
		}
	}
}

func TestPlanRefusesDivergedRedo(t *testing.T) {
	snapshot := repo.Snapshot{Root: "/repo", Branch: repo.Branch{Name: "main", OID: newHead}}
	request := Request{Repository: "/repo", Kind: "undo commit", Ref: "main", OldHead: oldHead, NewHead: newHead, PostSnapshotHash: "stale"}
	if _, err := Plan(request, snapshot); !errors.Is(err, ErrDiverged) {
		t.Fatalf("error = %v, want ErrDiverged", err)
	}
}

func TestExecuteRedoReplaysSuccessfulSoftUndo(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	write := func(value string) {
		if err := os.WriteFile(filepath.Join(dir, "file"), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("one\n")
	if _, err := runner.Stage(ctx, []byte("file")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("one\n")}); err != nil {
		t.Fatal(err)
	}
	first := strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout))
	write("two\n")
	if _, err := runner.Stage(ctx, []byte("file")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("two\n")}); err != nil {
		t.Fatal(err)
	}
	second := strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout))
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "reset", "--soft", first); err != nil {
		t.Fatal(err)
	}
	afterUndo, err := git.Snapshot(ctx, discovery, 2)
	if err != nil {
		t.Fatal(err)
	}
	outcome := Execute(ctx, runner, Request{Repository: discovery.Root, Kind: "undo commit", Ref: "main", OldHead: second, NewHead: first, PostSnapshotHash: undo.SnapshotFingerprint(afterUndo), Discovery: discovery, Generation: 2})
	if outcome.Err != nil {
		t.Fatalf("redo outcome = %#v", outcome)
	}
	if got := strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout)); got != second {
		t.Fatalf("HEAD after redo = %s, want %s", got, second)
	}
}

func TestExecuteRefusesRedoAfterExternalCommit(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	write := func(name, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(name, value, message string) string {
		t.Helper()
		write(name, value)
		if _, err := runner.Stage(ctx, []byte(name)); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout))
	}
	base := commit("file", "base", "base")
	undone := commit("file", "undone", "undone")
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "reset", "--soft", base); err != nil {
		t.Fatal(err)
	}
	afterUndo, err := git.Snapshot(ctx, discovery, 1)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{
		Repository: discovery.Root, Kind: "undo commit", Ref: "main", OldHead: undone,
		NewHead: base, PostSnapshotHash: undo.SnapshotFingerprint(afterUndo),
		Discovery: discovery, Generation: 1,
	}
	externalHead := commit("external", "new work", "external change")
	before := mustRun(t, runner, ctx, "status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all")
	outcome := Execute(ctx, runner, request)
	if !errors.Is(outcome.Err, ErrDiverged) {
		t.Fatalf("redo after external commit error = %v, want ErrDiverged", outcome.Err)
	}
	if got := strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout)); got != externalHead {
		t.Fatalf("HEAD after refused redo = %s, want external commit %s", got, externalHead)
	}
	after := mustRun(t, runner, ctx, "status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all")
	if string(after.Stdout) != string(before.Stdout) {
		t.Fatalf("repository status changed after refused redo:\nbefore=%q\nafter=%q", before.Stdout, after.Stdout)
	}
}

func mustRun(t *testing.T, runner git.Runner, ctx context.Context, args ...string) git.Result {
	t.Helper()
	result, err := runner.Run(ctx, args...)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// Keep the test independent from policy internals while exercising the same
// deterministic status representation used by production requests.
func snapshotFingerprint(snapshot repo.Snapshot) string {
	return undo.SnapshotFingerprint(snapshot)
}
