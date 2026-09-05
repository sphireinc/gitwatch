package undo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

func TestExecuteUndoImmediatelyPreservesWorktreeContent(t *testing.T) {
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
	write("base\n")
	if _, err := runner.Stage(ctx, []byte("file")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("base\n")}); err != nil {
		t.Fatal(err)
	}
	oldResult, err := runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	oldHead := strings.TrimSpace(string(oldResult.Stdout))
	write("committed\n")
	if _, err := runner.Stage(ctx, []byte("file")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("change\n")}); err != nil {
		t.Fatal(err)
	}
	write("local follow-up\n")
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := git.Snapshot(ctx, discovery, 1)
	if err != nil {
		t.Fatal(err)
	}
	outcome := Execute(ctx, runner, Request{Repository: discovery.Root, Kind: "commit", Ref: "main", OldHead: oldHead, NewHead: snapshot.Branch.OID, PostSnapshotHash: SnapshotFingerprint(snapshot), Discovery: discovery, Generation: 1})
	if outcome.Err != nil {
		t.Fatalf("undo outcome = %#v", outcome)
	}
	if got := strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout)); got != oldHead {
		t.Fatalf("HEAD after undo = %s, want %s", got, oldHead)
	}
	content, err := os.ReadFile(filepath.Join(dir, "file"))
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content); got != "local follow-up\n" {
		t.Fatalf("worktree after undo = %q", got)
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

const (
	oldHead = "1111111111111111111111111111111111111111"
	newHead = "2222222222222222222222222222222222222222"
)

func TestPlanAllowsOnlyMatchingSoftCommitUndo(t *testing.T) {
	snapshot := repo.Snapshot{Root: "/repo", Branch: repo.Branch{Name: "main", OID: newHead}, Entries: []repo.Entry{{Path: repo.Path("work.txt"), Staged: true}}}
	request := Request{Repository: "/repo", Kind: "commit", Ref: "main", OldHead: oldHead, NewHead: newHead}
	request.PostSnapshotHash = SnapshotFingerprint(snapshot)
	args, err := Plan(request, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 3 || args[0] != "reset" || args[1] != "--soft" || args[2] != oldHead {
		t.Fatalf("undo args = %#v", args)
	}
}

func TestPlanRefusesDivergenceAndActiveOperations(t *testing.T) {
	snapshot := repo.Snapshot{Root: "/repo", Branch: repo.Branch{Name: "main", OID: newHead}}
	request := Request{Repository: "/repo", Kind: "commit", Ref: "main", OldHead: oldHead, NewHead: newHead, PostSnapshotHash: SnapshotFingerprint(snapshot)}
	for name, mutate := range map[string]func(*repo.Snapshot){
		"head": func(value *repo.Snapshot) { value.Branch.OID = oldHead },
		"content": func(value *repo.Snapshot) {
			value.Entries = []repo.Entry{{Path: repo.Path("changed.txt"), Unstaged: true}}
		},
		"active operation": func(value *repo.Snapshot) { var active sequencer.State; value.Operation = &active },
	} {
		candidate := snapshot.Clone()
		mutate(&candidate)
		if _, err := Plan(request, candidate); err == nil {
			t.Fatalf("%s divergence was accepted", name)
		}
	}
}

func TestExecuteRefusesUnrelatedCommitAfterRecordedOperation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("file")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("one\n")}); err != nil {
		t.Fatal(err)
	}
	old := strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout))
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("file")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("two\n")}); err != nil {
		t.Fatal(err)
	}
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := git.Snapshot(ctx, discovery, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "other"), []byte("unrelated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("other")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("unrelated\n")}); err != nil {
		t.Fatal(err)
	}
	outcome := Execute(ctx, runner, Request{
		Repository: discovery.Root, Kind: "commit", Ref: "main", OldHead: old,
		NewHead: recorded.Branch.OID, PostSnapshotHash: SnapshotFingerprint(recorded),
		Discovery: discovery, Generation: 2,
	})
	if !errors.Is(outcome.Err, ErrDiverged) {
		t.Fatalf("unrelated commit outcome = %v, want ErrDiverged", outcome.Err)
	}
}

func TestPlanRefusesForeignRepository(t *testing.T) {
	snapshot := repo.Snapshot{Root: "/repo-b", Branch: repo.Branch{Name: "main", OID: newHead}}
	_, err := Plan(Request{
		Repository: "/repo-a", Kind: "commit", Ref: "main", OldHead: oldHead,
		NewHead: newHead, PostSnapshotHash: SnapshotFingerprint(snapshot),
	}, snapshot)
	if !errors.Is(err, ErrRepositoryMismatch) {
		t.Fatalf("foreign repository error = %v, want ErrRepositoryMismatch", err)
	}
}
