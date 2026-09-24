package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/rebase"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

func TestDetectOperationStateNoOperation(t *testing.T) {
	runner, discovery := operationFixture(t)
	got, err := DetectOperationState(context.Background(), discovery, 9)
	if err != nil {
		t.Fatal(err)
	}
	if got.Found {
		t.Fatalf("unexpected operation state: %#v", got)
	}
	_ = runner
}

func TestRebaseStoppedAtEditUsesGitCompletedTodoAction(t *testing.T) {
	for _, test := range []struct {
		name      string
		done      string
		stopped   string
		wantPause bool
	}{
		{name: "edit action", done: "pick aaa first\nedit bbb second\n", stopped: "bbb", wantPause: true},
		{name: "conflicted pick", done: "pick aaa first\n", stopped: "aaa", wantPause: false},
		{name: "different stopped commit", done: "edit aaa first\n", stopped: "bbb", wantPause: false},
		{name: "missing stopped sha", done: "edit aaa first\n", wantPause: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := rebaseStoppedAtEdit(test.done, test.stopped); got != test.wantPause {
				t.Fatalf("rebaseStoppedAtEdit(%q, %q) = %v, want %v", test.done, test.stopped, got, test.wantPause)
			}
		})
	}
}

func TestDetectOperationStateMergeCherryPickRevertAndRebase(t *testing.T) {
	tests := []struct {
		name  string
		kind  sequencer.Kind
		start func(*testing.T, Runner, string)
		stop  func(*testing.T, Runner)
	}{
		{"merge", sequencer.KindMerge, startMerge, abortMerge},
		{"cherry-pick", sequencer.KindCherryPick, startCherryPick, abortCherryPick},
		{"revert", sequencer.KindRevert, startRevert, abortRevert},
		{"rebase", sequencer.KindRebase, startRebase, abortRebase},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner, discovery := operationFixture(t)
			test.start(t, runner, discovery.Root)
			defer test.stop(t, runner)
			got, err := DetectOperationState(context.Background(), discovery, 17)
			if err != nil {
				t.Fatal(err)
			}
			if !got.Found || got.State.Kind() != test.kind || got.State.RepositoryID() != sequencer.RepositoryID(discovery.Root) || got.State.Generation() != 17 {
				t.Fatalf("detected state = %#v", got)
			}
			if got.State.Phase() != sequencer.PhaseActive {
				t.Fatalf("phase = %s", got.State.Phase())
			}
			if test.kind == sequencer.KindRebase && got.State.Details().Rebase.EditStopped {
				t.Fatal("conflict rebase was incorrectly identified as an edit-stop")
			}
			if got.State.HeadCurrent() == "" || got.State.Details() == (sequencer.Details{}) {
				t.Fatalf("incomplete operation projection: head=%q details=%#v", got.State.HeadCurrent(), got.State.Details())
			}
		})
	}
}

func TestDetectOperationStateSurvivesRunnerReconstructionDuringRebase(t *testing.T) {
	runner, discovery := operationFixture(t)
	startRebase(t, runner, discovery.Root)

	// Simulate gitwatch restarting: no in-memory operation state is reused.
	restartedRunner := NewRunner(discovery.Root)
	restartedDiscovery, err := Discover(context.Background(), restartedRunner.Dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DetectOperationState(context.Background(), restartedDiscovery, 52)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found || got.State.Kind() != sequencer.KindRebase || got.State.RepositoryID() != sequencer.RepositoryID(discovery.Root) || got.State.Generation() != 52 {
		t.Fatalf("restarted rebase state = %#v", got)
	}
	if got.State.Phase() != sequencer.PhaseActive || got.State.Details().Rebase == nil {
		t.Fatalf("restarted rebase projection = phase=%s details=%#v", got.State.Phase(), got.State.Details())
	}
	if _, err := restartedRunner.OperationLifecycle(context.Background(), sequencer.KindRebase, "abort"); err != nil {
		t.Fatal(err)
	}
}

func TestDetectOperationStateReportsCherryPickProgress(t *testing.T) {
	runner, discovery := operationFixture(t)
	if _, err := runner.Run(context.Background(), "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, discovery.Root, "feature one\n", "feature one")
	first := rev(t, runner, "HEAD")
	commitFile(t, runner, discovery.Root, "feature two\n", "feature two")
	second := rev(t, runner, "HEAD")
	if _, err := runner.Run(context.Background(), "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, discovery.Root, "main\n", "main")
	if _, err := runner.Run(context.Background(), "checkout", "feature"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "cherry-pick", first, second); err == nil {
		t.Fatal("cherry-pick unexpectedly completed")
	}
	defer func() { _, _ = runner.Run(context.Background(), "cherry-pick", "--abort") }()

	got, err := DetectOperationState(context.Background(), discovery, 53)
	if err != nil {
		t.Fatal(err)
	}
	details := got.State.Details().CherryPick
	if !got.Found || got.State.Kind() != sequencer.KindCherryPick || details == nil || len(details.Commits) != 2 || details.CurrentIndex != 0 || got.State.Remaining() != 2 {
		t.Fatalf("cherry-pick progress = found=%v kind=%s details=%#v remaining=%d", got.Found, got.State.Kind(), details, got.State.Remaining())
	}
}

func TestDetectCherryPickMiddleConflictReconstructsAppliedResultAfterRestart(t *testing.T) {
	ctx := context.Background()
	runner, discovery := operationFixture(t)
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	commitNamed := func(path, content, message string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(discovery.Root, path), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Run(ctx, "add", "--", path); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Run(ctx, "commit", "-m", message); err != nil {
			t.Fatal(err)
		}
		return rev(t, runner, "HEAD")
	}
	first := commitNamed("first.txt", "first\n", "first")
	commitFile(t, runner, discovery.Root, "feature\n", "second")
	second := rev(t, runner, "HEAD")
	third := commitNamed("third.txt", "third\n", "third")
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, discovery.Root, "main\n", "main")
	original := rev(t, runner, "HEAD")
	if _, err := runner.Run(ctx, "cherry-pick", first, second, third); err == nil {
		t.Fatal("expected middle-commit conflict")
	}
	t.Cleanup(func() { _, _ = runner.Run(context.Background(), "cherry-pick", "--abort") })
	restarted, err := Discover(ctx, discovery.Root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DetectOperationState(ctx, restarted, 54)
	if err != nil {
		t.Fatal(err)
	}
	state := got.State
	details := state.Details().CherryPick
	applied := rev(t, NewRunner(discovery.Root), "HEAD")
	if !got.Found || state.Kind() != sequencer.KindCherryPick || state.HeadBefore() != original || state.HeadCurrent() != applied || state.Completed() != 1 || state.Remaining() != 2 || details == nil || len(details.Commits) != 3 || len(details.Completed) != 1 || details.Completed[0] != applied || details.Commits[0] != applied || details.CurrentIndex != 1 || !strings.HasPrefix(second, details.Commits[1]) {
		t.Fatalf("restarted middle-conflict state = found=%v state=%#v details=%#v", got.Found, state, details)
	}
}

func TestDetectCherryPickFirstAndLastConflictAfterRestart(t *testing.T) {
	for _, conflictIndex := range []int{0, 2} {
		t.Run(fmt.Sprintf("conflict-%d", conflictIndex+1), func(t *testing.T) {
			ctx := context.Background()
			runner, discovery := operationFixture(t)
			if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
				t.Fatal(err)
			}
			commits := make([]string, 3)
			for index := range commits {
				path := fmt.Sprintf("selected-%d.txt", index)
				if index == conflictIndex {
					path = "file.txt"
				}
				if err := os.WriteFile(filepath.Join(discovery.Root, path), []byte(fmt.Sprintf("feature-%d\n", index)), 0o644); err != nil {
					t.Fatal(err)
				}
				if _, err := runner.Run(ctx, "add", "--", path); err != nil {
					t.Fatal(err)
				}
				if _, err := runner.Run(ctx, "commit", "-m", fmt.Sprintf("selected %d", index)); err != nil {
					t.Fatal(err)
				}
				commits[index] = rev(t, runner, "HEAD")
			}
			if _, err := runner.Run(ctx, "switch", "main"); err != nil {
				t.Fatal(err)
			}
			commitFile(t, runner, discovery.Root, "main\n", "main")
			original := rev(t, runner, "HEAD")
			if _, err := runner.Run(ctx, "cherry-pick", commits[0], commits[1], commits[2]); err == nil {
				t.Fatal("expected cherry-pick conflict")
			}
			t.Cleanup(func() { _, _ = runner.Run(context.Background(), "cherry-pick", "--abort") })
			restarted, err := Discover(ctx, discovery.Root)
			if err != nil {
				t.Fatal(err)
			}
			got, err := DetectOperationState(ctx, restarted, 55)
			if err != nil {
				t.Fatal(err)
			}
			state := got.State
			details := state.Details().CherryPick
			if !got.Found || state.Kind() != sequencer.KindCherryPick || state.HeadBefore() != original || state.Completed() != conflictIndex || state.Remaining() != 3-conflictIndex || details == nil || len(details.Commits) != 3 || details.CurrentIndex != conflictIndex || !strings.HasPrefix(commits[conflictIndex], details.Commits[conflictIndex]) {
				t.Fatalf("restarted cherry-pick state = found=%v state=%#v details=%#v", got.Found, state, details)
			}
			if _, err := NewRunner(discovery.Root).OperationLifecycle(ctx, sequencer.KindCherryPick, "skip"); err != nil {
				t.Fatal(err)
			}
			finished, err := DetectOperationState(ctx, restarted, 56)
			if err != nil {
				t.Fatal(err)
			}
			if finished.Found {
				t.Fatalf("cherry-pick still active after skip: %#v", finished)
			}
			for index := range commits {
				if index == conflictIndex {
					continue
				}
				path := filepath.Join(discovery.Root, fmt.Sprintf("selected-%d.txt", index))
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("selected commit %d missing after skip: %v", index, err)
				}
			}
		})
	}
}

func TestDetectOperationStateReportsRevertProgress(t *testing.T) {
	runner, discovery := operationFixture(t)
	commitFile(t, runner, discovery.Root, "first\n", "first")
	first := rev(t, runner, "HEAD")
	commitFile(t, runner, discovery.Root, "second\n", "second")
	if _, err := runner.Run(context.Background(), "revert", "--no-edit", first); err == nil {
		t.Fatal("revert unexpectedly completed")
	}
	defer func() { _, _ = runner.Run(context.Background(), "revert", "--abort") }()

	got, err := DetectOperationState(context.Background(), discovery, 54)
	if err != nil {
		t.Fatal(err)
	}
	details := got.State.Details().Revert
	if !got.Found || got.State.Kind() != sequencer.KindRevert || details == nil || len(details.Commits) != 1 || details.Commits[0] != first || got.State.Remaining() != 1 {
		t.Fatalf("revert progress = found=%v kind=%s details=%#v remaining=%d", got.Found, got.State.Kind(), details, got.State.Remaining())
	}
}

func TestDetectOperationStateBisect(t *testing.T) {
	runner, discovery := operationFixture(t)
	commitFile(t, runner, discovery.Root, "one\n", "one")
	good := rev(t, runner, "HEAD")
	commitFile(t, runner, discovery.Root, "two\n", "two")
	commitFile(t, runner, discovery.Root, "three\n", "three")
	bad := rev(t, runner, "HEAD")
	if _, err := runner.Run(context.Background(), "bisect", "start", bad, good); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = runner.Run(context.Background(), "bisect", "reset") }()
	got, err := DetectOperationState(context.Background(), discovery, 21)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found || got.State.Kind() != sequencer.KindBisect || got.State.Details().Bisect == nil {
		t.Fatalf("bisect state = %#v", got)
	}
}

func TestDetectOperationStateUsesResolvedLinkedWorktreeGitdir(t *testing.T) {
	runner, discovery := operationFixture(t)
	linked := filepath.Join(t.TempDir(), "linked")
	if _, err := runner.Run(context.Background(), "worktree", "add", "-b", "linked", "--", linked); err != nil {
		t.Fatal(err)
	}
	linkedDiscovery, err := Discover(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DetectOperationState(context.Background(), linkedDiscovery, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got.Found || !linkedDiscovery.Linked || filepath.Clean(linkedDiscovery.GitDir) == filepath.Join(linked, ".git") {
		t.Fatalf("linked discovery/state = %+v / %#v", linkedDiscovery, got)
	}
	_ = discovery
}

func TestDetectOperationStateDegradesUnknownSequencerMetadata(t *testing.T) {
	runner, discovery := operationFixture(t)
	result, err := runner.Run(context.Background(), "rev-parse", "--git-path", "sequencer")
	if err != nil {
		t.Fatal(err)
	}
	sequencerPath := filepath.Clean(string(result.Stdout[:len(result.Stdout)-1]))
	if !filepath.IsAbs(sequencerPath) {
		sequencerPath = filepath.Join(discovery.Root, sequencerPath)
	}
	if err := os.MkdirAll(sequencerPath, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := DetectOperationState(context.Background(), discovery, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Found || got.State.Kind() != sequencer.KindUnknown || len(got.Diagnostics) != 1 {
		t.Fatalf("unknown operation state = %#v", got)
	}
}

func TestSnapshotCarriesOperationAtSameGeneration(t *testing.T) {
	runner, discovery := operationFixture(t)
	startMerge(t, runner, discovery.Root)
	defer abortMerge(t, runner)
	snapshot, err := Snapshot(context.Background(), discovery, 33)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Generation != 33 || snapshot.Operation == nil || snapshot.Operation.Kind() != sequencer.KindMerge || snapshot.Operation.Generation() != snapshot.Generation {
		t.Fatalf("snapshot operation=%#v generation=%d", snapshot.Operation, snapshot.Generation)
	}
	clone := snapshot.Clone()
	if clone.Operation == nil || clone.Operation == snapshot.Operation {
		t.Fatal("snapshot clone did not copy operation projection")
	}
}

func TestOperationLifecycleSkipsRebase(t *testing.T) {
	runner, discovery := operationFixture(t)
	startRebase(t, runner, discovery.Root)
	if _, err := runner.OperationLifecycle(context.Background(), sequencer.KindRebase, "skip"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := Snapshot(context.Background(), discovery, 34)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Operation != nil || snapshot.Branch.Name != "feature" {
		t.Fatalf("post-skip snapshot = %+v", snapshot)
	}
	content, err := os.ReadFile(filepath.Join(discovery.Root, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "main\n" {
		t.Fatalf("post-skip content = %q", content)
	}
}

func TestInteractiveRebaseEditStopSurvivesRestartAndCanSkip(t *testing.T) {
	restartedRunner, restartedDiscovery, _, _ := startInteractiveEditStop(t)
	if _, err := restartedRunner.OperationLifecycle(context.Background(), sequencer.KindRebase, "skip"); err != nil {
		t.Fatalf("skip edit-stopped commit: %v", err)
	}
	snapshot, err := Snapshot(context.Background(), restartedDiscovery, 56)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Operation != nil || snapshot.Branch.Name != "feature" {
		t.Fatalf("post-skip snapshot = %+v", snapshot)
	}
	content, err := os.ReadFile(filepath.Join(restartedDiscovery.Root, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "second\n" {
		t.Fatalf("post-skip content = %q", content)
	}
}

func TestInteractiveRebaseEditStopCanContinue(t *testing.T) {
	restartedRunner, restartedDiscovery, _, _ := startInteractiveEditStop(t)
	if _, err := restartedRunner.OperationLifecycle(context.Background(), sequencer.KindRebase, "continue"); err != nil {
		t.Fatalf("continue edit-stopped rebase: %v", err)
	}
	snapshot, err := Snapshot(context.Background(), restartedDiscovery, 57)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Operation != nil || snapshot.Branch.Name != "feature" {
		t.Fatalf("post-continue snapshot = %+v", snapshot)
	}
	content, err := os.ReadFile(filepath.Join(restartedDiscovery.Root, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "second\n" {
		t.Fatalf("post-continue content = %q", content)
	}
}

func TestInteractiveRebaseEditStopCanAmendAndContinue(t *testing.T) {
	restartedRunner, restartedDiscovery, originalFirst, _ := startInteractiveEditStop(t)
	if _, err := restartedRunner.Run(context.Background(), "commit", "--amend", "-m", "amended first"); err != nil {
		t.Fatalf("amend edit-stopped commit: %v", err)
	}
	amendedFirst := rev(t, restartedRunner, "HEAD")
	if amendedFirst == originalFirst {
		t.Fatal("amended commit retained its original object ID")
	}
	messageResult, err := restartedRunner.Run(context.Background(), "log", "-1", "--format=%s")
	if err != nil {
		t.Fatal(err)
	}
	message := strings.TrimSpace(string(messageResult.Stdout))
	if message != "amended first" {
		t.Fatalf("amended commit subject = %q", message)
	}
	if _, err := restartedRunner.OperationLifecycle(context.Background(), sequencer.KindRebase, "continue"); err != nil {
		t.Fatalf("continue amended rebase: %v", err)
	}
	snapshot, err := Snapshot(context.Background(), restartedDiscovery, 58)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Operation != nil || snapshot.Branch.Name != "feature" || rev(t, restartedRunner, "HEAD~1") != amendedFirst {
		t.Fatalf("post-amend snapshot = %+v; HEAD~1=%s want %s", snapshot, rev(t, restartedRunner, "HEAD~1"), amendedFirst)
	}
	content, err := os.ReadFile(filepath.Join(restartedDiscovery.Root, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "second\n" {
		t.Fatalf("post-amend content = %q", content)
	}
}

func TestInteractiveRebaseEditStopCanAbortAfterRestart(t *testing.T) {
	restartedRunner, restartedDiscovery, _, originalHead := startInteractiveEditStop(t)
	if _, err := restartedRunner.OperationLifecycle(context.Background(), sequencer.KindRebase, "abort"); err != nil {
		t.Fatalf("abort edit-stopped rebase: %v", err)
	}
	snapshot, err := Snapshot(context.Background(), restartedDiscovery, 59)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Operation != nil || snapshot.Branch.Name != "feature" || rev(t, restartedRunner, "HEAD") != originalHead {
		t.Fatalf("post-abort snapshot = %+v; HEAD=%s want %s", snapshot, rev(t, restartedRunner, "HEAD"), originalHead)
	}
	content, err := os.ReadFile(filepath.Join(restartedDiscovery.Root, "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "second\n" {
		t.Fatalf("post-abort content = %q", content)
	}
}

func startInteractiveEditStop(t *testing.T) (Runner, Discovery, string, string) {
	t.Helper()
	runner, discovery := operationFixture(t)
	base := rev(t, runner, "HEAD")
	if _, err := runner.Run(context.Background(), "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, discovery.Root, "first\n", "first")
	first := rev(t, runner, "HEAD")
	commitFile(t, runner, discovery.Root, "second\n", "second")
	second := rev(t, runner, "HEAD")

	plan, err := rebase.Parse(fmt.Sprintf("edit %s first\npick %s second\n", first, second))
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	runner.Env = append(runner.Env,
		"GITWATCH_TEST_SEQUENCE_EDITOR=1",
		"GITWATCH_TEST_SEQUENCE_PLAN="+plan.Render(),
	)
	outcome, err := runner.StartInteractiveRebase(context.Background(), RebaseRequest{
		Base:   base,
		Plan:   plan,
		Editor: strconv.Quote(executable) + " -test.run=^TestSequenceEditorHelper$ --",
	})
	if err != nil {
		t.Fatalf("start edit-stop rebase: %v (outcome=%+v)", err, outcome)
	}
	if !outcome.Paused || outcome.State == nil || outcome.State.Kind() != sequencer.KindRebase {
		t.Fatalf("edit-stop outcome = %+v", outcome)
	}
	if outcome.State.CurrentCommit() != first || outcome.State.Remaining() != 1 || outcome.State.Completed() != 1 {
		t.Fatalf("edit-stop progress = current=%q completed=%d remaining=%d", outcome.State.CurrentCommit(), outcome.State.Completed(), outcome.State.Remaining())
	}
	if details := outcome.State.Details().Rebase; details == nil || !details.EditStopped {
		t.Fatalf("edit-stop was not captured in Git-derived details: %+v", details)
	}

	// Restart with a fresh runner and reconstruct the stopped state from Git.
	restartedRunner := NewRunner(discovery.Root)
	restartedRunner.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null"}
	restartedDiscovery, err := Discover(context.Background(), restartedRunner.Dir)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := DetectOperationState(context.Background(), restartedDiscovery, 55)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered.Found || recovered.State.Kind() != sequencer.KindRebase || recovered.State.CurrentCommit() != first || recovered.State.Completed() != 1 || recovered.State.Remaining() != 1 {
		t.Fatalf("recovered edit-stop = found=%v kind=%s current=%q completed=%d remaining=%d", recovered.Found, recovered.State.Kind(), recovered.State.CurrentCommit(), recovered.State.Completed(), recovered.State.Remaining())
	}
	if details := recovered.State.Details().Rebase; details == nil || !details.EditStopped {
		t.Fatalf("restarted edit-stop details = %+v", details)
	}
	return restartedRunner, restartedDiscovery, first, second
}

// TestSequenceEditorHelper runs only as the GIT_SEQUENCE_EDITOR subprocess
// for TestInteractiveRebaseEditStopSurvivesRestartAndCanSkip.
func TestSequenceEditorHelper(t *testing.T) {
	if os.Getenv("GITWATCH_TEST_SEQUENCE_EDITOR") != "1" {
		return
	}
	if len(os.Args) < 2 {
		t.Fatal("sequence editor did not receive Git's todo path")
	}
	todoPath := os.Args[len(os.Args)-1]
	if err := os.WriteFile(todoPath, []byte(os.Getenv("GITWATCH_TEST_SEQUENCE_PLAN")), 0o600); err != nil {
		t.Fatal(err)
	}
}

func operationFixture(t *testing.T) (Runner, Discovery) {
	t.Helper()
	dir := t.TempDir()
	runner := NewRunner(dir)
	runner.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null"}
	if _, err := runner.Run(context.Background(), "init", "-b", "main", "--", dir); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"config", "user.name", "gitwatch-test"}, {"config", "user.email", "gitwatch@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	commitFile(t, runner, dir, "base\n", "base")
	discovery, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return runner, discovery
}

func commitFile(t *testing.T, runner Runner, dir, content, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "add", "--", "file.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "commit", "-m", message); err != nil {
		t.Fatal(err)
	}
}

func rev(t *testing.T, runner Runner, ref string) string {
	t.Helper()
	result, err := runner.Run(context.Background(), "rev-parse", "--verify", ref)
	if err != nil {
		t.Fatal(err)
	}
	return string(result.Stdout[:len(result.Stdout)-1])
}

func startMerge(t *testing.T, runner Runner, dir string) {
	if _, err := runner.Run(context.Background(), "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, dir, "feature\n", "feature")
	if _, err := runner.Run(context.Background(), "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, dir, "main\n", "main")
	if _, err := runner.Run(context.Background(), "merge", "feature"); err == nil {
		t.Fatal("merge unexpectedly completed")
	}
}

func abortMerge(t *testing.T, runner Runner) {
	if _, err := runner.Run(context.Background(), "merge", "--abort"); err != nil {
		t.Fatal(err)
	}
}

func startCherryPick(t *testing.T, runner Runner, dir string) {
	if _, err := runner.Run(context.Background(), "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, dir, "feature\n", "feature")
	feature := rev(t, runner, "HEAD")
	if _, err := runner.Run(context.Background(), "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, dir, "main\n", "main")
	if _, err := runner.Run(context.Background(), "cherry-pick", feature); err == nil {
		t.Fatal("cherry-pick unexpectedly completed")
	}
}

func abortCherryPick(t *testing.T, runner Runner) {
	if _, err := runner.Run(context.Background(), "cherry-pick", "--abort"); err != nil {
		t.Fatal(err)
	}
}

func startRevert(t *testing.T, runner Runner, dir string) {
	commitFile(t, runner, dir, "first\n", "first")
	first := rev(t, runner, "HEAD")
	commitFile(t, runner, dir, "second\n", "second")
	if _, err := runner.Run(context.Background(), "revert", first); err == nil {
		t.Fatal("revert unexpectedly completed")
	}
}

func abortRevert(t *testing.T, runner Runner) {
	if _, err := runner.Run(context.Background(), "revert", "--abort"); err != nil {
		t.Fatal(err)
	}
}

func startRebase(t *testing.T, runner Runner, dir string) {
	if _, err := runner.Run(context.Background(), "checkout", "-b", "feature"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, dir, "feature\n", "feature")
	if _, err := runner.Run(context.Background(), "checkout", "main"); err != nil {
		t.Fatal(err)
	}
	commitFile(t, runner, dir, "main\n", "main")
	if _, err := runner.Run(context.Background(), "checkout", "feature"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "rebase", "main"); err == nil {
		t.Fatal("rebase unexpectedly completed")
	}
}

func abortRebase(t *testing.T, runner Runner) {
	if _, err := runner.Run(context.Background(), "rebase", "--abort"); err != nil && !errors.Is(err, ErrCommandFailed) {
		t.Fatal(err)
	}
}
