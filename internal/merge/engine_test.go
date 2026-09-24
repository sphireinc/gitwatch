package merge

import (
	"context"
	"errors"
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
	if outcome.Err != nil || outcome.Paused || outcome.Snapshot == nil || outcome.Snapshot.Branch.Name != "main" {
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

func TestNoFastForwardMergeCreatesTwoParentCommit(t *testing.T) {
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

	outcome := (Engine{Runner: runner, Repository: dir}).Execute(ctx, Request{
		Repository: dir,
		Source:     "feature",
		Strategy:   NoFastForward,
		Message:    "integrate feature",
	})
	if outcome.Err != nil || outcome.Paused || outcome.Snapshot == nil || outcome.Snapshot.Operation != nil {
		t.Fatalf("no-ff outcome = %#v", outcome)
	}
	parents, err := runner.Run(ctx, "rev-list", "--parents", "-n", "1", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got := len(strings.Fields(string(parents.Stdout))); got != 3 {
		t.Fatalf("merge commit has %d fields, want commit and two parents: %q", got, parents.Stdout)
	}
	message, err := runner.Run(ctx, "log", "-1", "--format=%s")
	if err != nil || strings.TrimSpace(string(message.Stdout)) != "integrate feature" {
		t.Fatalf("merge message = %q, err=%v", message.Stdout, err)
	}
}

func TestSquashMergeLeavesStagedChangesWithoutMergeCommit(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := setupMergeRepository(t, dir)
	baseHead := revMerge(t, runner, "HEAD")
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	writeMergeFile(t, runner, dir, "feature\n", "feature")
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}

	outcome := (Engine{Runner: runner, Repository: dir}).Execute(ctx, Request{
		Repository: dir,
		Source:     "feature",
		Strategy:   Squash,
	})
	if outcome.Err != nil || outcome.Paused || outcome.Snapshot == nil || outcome.Snapshot.Operation != nil {
		t.Fatalf("squash outcome = %#v", outcome)
	}
	if got := revMerge(t, runner, "HEAD"); got != baseHead {
		t.Fatalf("squash created a commit: HEAD=%s, want unchanged %s", got, baseHead)
	}
	if outcome.Snapshot.Counts.Staged == 0 {
		t.Fatalf("squash did not return staged state: %+v", outcome.Snapshot.Counts)
	}
	if _, err := os.Stat(filepath.Join(dir, "shared")); err != nil {
		t.Fatal(err)
	}
	status, err := runner.Run(ctx, "diff", "--cached", "--name-only")
	if err != nil || strings.TrimSpace(string(status.Stdout)) != "shared" {
		t.Fatalf("squash staged paths = %q, err=%v", status.Stdout, err)
	}
}

func TestFastForwardOnlyRefusesDivergedBranchesWithoutMutation(t *testing.T) {
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
	mainHead := revMerge(t, runner, "HEAD")

	outcome := (Engine{Runner: runner, Repository: dir}).Execute(ctx, Request{
		Repository: dir,
		Source:     "feature",
		Strategy:   FastForwardOnly,
	})
	if outcome.Err == nil || outcome.Paused || outcome.Snapshot == nil || outcome.Snapshot.Operation != nil {
		t.Fatalf("ff-only refusal = %#v", outcome)
	}
	if got := revMerge(t, runner, "HEAD"); got != mainHead {
		t.Fatalf("ff-only refusal changed HEAD: got %s, want %s", got, mainHead)
	}
	content, err := os.ReadFile(filepath.Join(dir, "shared"))
	if err != nil || string(content) != "main\n" {
		t.Fatalf("worktree after ff-only refusal = %q, err=%v", content, err)
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

func TestMergeRejectsStaleRepositoryGenerationBeforeGit(t *testing.T) {
	engine := Engine{Repository: "repo", Generation: 12}
	outcome := engine.Execute(context.Background(), Request{Repository: "repo", Generation: 11, Source: "feature"})
	if !errors.Is(outcome.Err, ErrStaleGeneration) {
		t.Fatalf("stale generation outcome = %#v", outcome)
	}
}

func TestMergeRejectsCurrentBranchSource(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := setupMergeRepository(t, dir)
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	engine := Engine{Runner: runner, Discovery: discovery, Repository: dir}
	outcome := engine.Execute(ctx, Request{Repository: dir, Source: "main"})
	if !errors.Is(outcome.Err, ErrCurrentBranch) {
		t.Fatalf("current branch outcome = %#v", outcome)
	}
}

func TestMergeRejectsSourceCheckedOutInLinkedWorktree(t *testing.T) {
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
	linked := filepath.Join(t.TempDir(), "feature-worktree")
	if _, err := runner.Run(ctx, "worktree", "add", linked, "feature"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = runner.Run(ctx, "worktree", "remove", "--force", linked) }()
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	outcome := (Engine{Runner: runner, Discovery: discovery, Repository: dir}).Execute(ctx, Request{Repository: dir, Source: "feature"})
	if !errors.Is(outcome.Err, ErrSourceOccupied) {
		t.Fatalf("occupied source outcome = %#v", outcome)
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
	if outcome.Snapshot == nil || outcome.Snapshot.Operation == nil || outcome.Snapshot.Counts.Conflicted == 0 {
		t.Fatalf("conflict snapshot = %#v", outcome.Snapshot)
	}
	aborted := engine.Abort(ctx)
	if aborted.Err != nil || aborted.Paused || aborted.Snapshot == nil || aborted.Snapshot.Operation != nil {
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
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}, {"config", "commit.gpgsign", "false"}} {
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

func revMerge(t *testing.T, runner git.Runner, ref string) string {
	t.Helper()
	result, err := runner.Run(context.Background(), "rev-parse", "--verify", ref)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(result.Stdout))
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
