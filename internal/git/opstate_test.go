package git

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

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
			if got.State.HeadCurrent() == "" || got.State.Details() == (sequencer.Details{}) {
				t.Fatalf("incomplete operation projection: head=%q details=%#v", got.State.HeadCurrent(), got.State.Details())
			}
		})
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
