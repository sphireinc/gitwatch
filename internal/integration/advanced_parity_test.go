package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/bisect"
	"github.com/sphireinc/git-watch/internal/blame"
	"github.com/sphireinc/git-watch/internal/cherrypick"
	"github.com/sphireinc/git-watch/internal/compare"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/merge"
	"github.com/sphireinc/git-watch/internal/pathhistory"
	"github.com/sphireinc/git-watch/internal/reflog"
	"github.com/sphireinc/git-watch/internal/sequencer"
	"github.com/sphireinc/git-watch/internal/tags"
)

// TestAdvancedHistoryAndComparisonParityScenario keeps several read-only
// parity claims tied to one disposable repository with real Git state. The
// domain loaders must consume the same commits, refs, and paths that Git
// reports; mocked command output would not prove that contract.
func TestAdvancedHistoryAndComparisonParityScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "gitwatch-parity"},
		{"config", "user.email", "gitwatch-parity@example.com"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(root, "notes.txt")
	commit := func(content, message string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("notes.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}

	first := commit("one\ntwo\n", "initial notes")
	second := commit("one\nTWO\nthree\n", "expand notes")
	if _, err := runner.Run(ctx, "tag", "v-parity-lightweight", first); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "tag", "-a", "v-parity-annotated", "-m", "parity release", second); err != nil {
		t.Fatal(err)
	}
	tagsSnapshot, err := tags.Load(ctx, runner, tags.LoadRequest{Repository: root})
	if err != nil || len(tagsSnapshot.Tags) != 2 {
		t.Fatalf("tag parity snapshot = %#v, err=%v", tagsSnapshot, err)
	}
	if tagsSnapshot.Tags[0].Name != "v-parity-annotated" || tagsSnapshot.Tags[1].Name != "v-parity-lightweight" {
		t.Fatalf("tag ordering = %#v", tagsSnapshot.Tags)
	}

	reflogEntries, err := reflog.Load(ctx, runner, reflog.Request{Ref: "HEAD", Limit: 10})
	if err != nil || len(reflogEntries) < 2 {
		t.Fatalf("reflog parity entries = %#v, err=%v", reflogEntries, err)
	}

	blamePage, err := blame.LoadPage(ctx, runner, blame.Request{Path: "notes.txt", Start: 1, Limit: 10})
	if err != nil || len(blamePage.Lines) != 3 {
		t.Fatalf("blame parity page = %#v, err=%v", blamePage, err)
	}
	if string(blamePage.Lines[1].Content) != "TWO" {
		t.Fatalf("blame content = %q", blamePage.Lines[1].Content)
	}

	historyPage, err := pathhistory.LoadPage(ctx, runner, pathhistory.Request{Path: "notes.txt", Follow: true, Limit: 10})
	if err != nil || len(historyPage.Entries) < 2 {
		t.Fatalf("path history parity page = %#v, err=%v", historyPage, err)
	}

	comparison, err := compare.Compare(ctx, runner, compare.Request{Left: first, Right: second, MaxFiles: 10, MaxCommits: 10})
	if err != nil || len(comparison.Changes) != 1 || comparison.Changes[0].NewPath != "notes.txt" {
		t.Fatalf("comparison parity result = %#v, err=%v", comparison, err)
	}
	if !strings.Contains(comparison.Patch, "TWO") || !strings.Contains(comparison.Patch, "three") {
		t.Fatalf("comparison patch did not preserve changed content: %q", comparison.Patch)
	}
}

func TestBisectParityScenarioSurvivesFreshLoaderAndReset(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "gitwatch-bisect-parity"},
		{"config", "user.email", "gitwatch-bisect-parity@example.com"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(root, "signal.txt")
	commit := func(content, message string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("signal.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}

	good := commit("good\n", "good")
	commit("middle\n", "middle")
	bad := commit("bad\n", "bad")
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	started := bisect.Start(ctx, runner, bisect.StartRequest{Repository: discovery.Root, Generation: 1, Bad: bad, Good: good})
	if started.Err != nil || !started.State.Active {
		t.Fatalf("bisect start = %#v", started)
	}
	defer func() { _, _ = runner.Run(ctx, "bisect", "reset") }()

	// Reload through a fresh runner to prove that the state is reconstructed
	// from Git metadata rather than retained UI state.
	fresh := git.NewRunner(root)
	loaded, err := bisect.Load(ctx, fresh, discovery.Root, 2)
	if err != nil || !loaded.Active || loaded.Candidate == "" || len(loaded.Log) == 0 {
		t.Fatalf("fresh bisect state = %#v, err=%v", loaded, err)
	}
	marked := bisect.MarkCandidate(ctx, fresh, bisect.Request{Repository: discovery.Root, Generation: 3, Mark: bisect.Skip})
	if !marked.State.Active || len(marked.State.Log) == 0 {
		t.Fatalf("bisect skip lost active state: %#v", marked)
	}
	if reset := bisect.Reset(ctx, fresh, bisect.Request{Repository: discovery.Root, Generation: 4}); reset.Err != nil || reset.State.Active {
		t.Fatalf("bisect reset = %#v", reset)
	}
}

func TestCherryPickConflictResumeParityScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "gitwatch-cherry-pick-parity"},
		{"config", "user.email", "gitwatch-cherry-pick-parity@example.com"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(root, "conflict.txt")
	commit := func(content, message string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("conflict.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}

	commit("base\n", "base")
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	feature := commit("feature\n", "feature change")
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	commit("main\n", "main change")

	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	engine := cherrypick.Engine{Runner: runner, Discovery: discovery, Repository: root, Generation: 1}
	paused := engine.Execute(ctx, cherrypick.Request{Repository: root, Generation: 1, SHAs: []string{feature}})
	if paused.Err == nil || !paused.Paused || paused.State == nil || paused.Snapshot == nil || len(paused.Snapshot.Conflicts) == 0 {
		t.Fatalf("cherry-pick conflict = %#v", paused)
	}

	if err := os.WriteFile(path, []byte("resolved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("conflict.txt")); err != nil {
		t.Fatal(err)
	}
	continued := engine.Continue(ctx)
	if continued.Err != nil || continued.Paused || continued.Snapshot == nil {
		t.Fatalf("cherry-pick continue = %#v", continued)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "resolved\n" {
		t.Fatalf("resolved cherry-pick contents = %q, err=%v", contents, err)
	}
}

func TestMergeConflictResumeParityScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "gitwatch-merge-parity"},
		{"config", "user.email", "gitwatch-merge-parity@example.com"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(root, "merge.txt")
	commit := func(content, message string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("merge.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}

	commit("base\n", "base")
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	commit("feature\n", "feature change")
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	commit("main\n", "main change")

	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	engine := merge.Engine{Runner: runner, Discovery: discovery, Repository: root, Generation: 1}
	paused := engine.Execute(ctx, merge.Request{Repository: root, Generation: 1, Source: "feature", Strategy: merge.Regular})
	if paused.Err == nil || !paused.Paused || paused.State == nil || paused.Snapshot == nil || len(paused.Snapshot.Conflicts) == 0 {
		t.Fatalf("merge conflict = %#v", paused)
	}

	if err := os.WriteFile(path, []byte("resolved\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("merge.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.OperationLifecycle(ctx, sequencer.KindMerge, "continue"); err != nil {
		t.Fatal(err)
	}
	final, err := git.Snapshot(ctx, discovery, 2)
	if err != nil || final.Operation != nil || len(final.Conflicts) != 0 {
		t.Fatalf("completed merge snapshot = %#v, err=%v", final, err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "resolved\n" {
		t.Fatalf("resolved merge contents = %q, err=%v", contents, err)
	}
}

func TestRebaseConflictResumeParityScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "gitwatch-rebase-parity"},
		{"config", "user.email", "gitwatch-rebase-parity@example.com"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(root, "rebase.txt")
	commit := func(content, message string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("rebase.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}

	commit("base\n", "base")
	if _, err := runner.Run(ctx, "switch", "-c", "topic"); err != nil {
		t.Fatal(err)
	}
	topic := commit("topic\n", "topic change")
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	commit("main\n", "main change")
	if _, err := runner.Run(ctx, "switch", "topic"); err != nil {
		t.Fatal(err)
	}

	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	result, rebaseErr := runner.Run(ctx, "rebase", "main")
	if rebaseErr == nil {
		t.Fatalf("rebase unexpectedly completed: %#v", result)
	}
	operation, err := git.DetectOperationState(ctx, discovery, 1)
	if err != nil || !operation.Found || operation.State.Kind() != sequencer.KindRebase {
		t.Fatalf("rebase operation state = %#v, err=%v", operation, err)
	}
	snapshot, err := git.Snapshot(ctx, discovery, 1)
	if err != nil || len(snapshot.Conflicts) == 0 {
		t.Fatalf("rebase conflict snapshot = %#v, err=%v", snapshot, err)
	}
	if topic == "" {
		t.Fatal("topic commit was not created")
	}
	if err := os.WriteFile(path, []byte("resolved during rebase\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("rebase.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ContinueRebase(ctx); err != nil {
		t.Fatal(err)
	}
	final, err := git.Snapshot(ctx, discovery, 2)
	if err != nil || final.Operation != nil || len(final.Conflicts) != 0 {
		t.Fatalf("completed rebase snapshot = %#v, err=%v", final, err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "resolved during rebase\n" {
		t.Fatalf("rebased contents = %q, err=%v", contents, err)
	}
}

func TestRevertConflictResumeParityScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "gitwatch-revert-parity"},
		{"config", "user.email", "gitwatch-revert-parity@example.com"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(root, "revert.txt")
	commit := func(content, message string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("revert.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}

	commit("base\n", "base")
	first := commit("first\n", "first change")
	commit("second\n", "second change")
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, revertErr := runner.Revert(ctx, git.RevertRequest{Commits: []string{first}}); revertErr == nil {
		t.Fatal("revert unexpectedly completed without conflict")
	}
	operation, err := git.DetectOperationState(ctx, discovery, 1)
	if err != nil || !operation.Found || operation.State.Kind() != sequencer.KindRevert {
		t.Fatalf("revert operation state = %#v, err=%v", operation, err)
	}
	snapshot, err := git.Snapshot(ctx, discovery, 1)
	if err != nil || len(snapshot.Conflicts) == 0 {
		t.Fatalf("revert conflict snapshot = %#v, err=%v", snapshot, err)
	}
	if err := os.WriteFile(path, []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("revert.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.OperationLifecycle(ctx, sequencer.KindRevert, "continue"); err != nil {
		t.Fatal(err)
	}
	final, err := git.Snapshot(ctx, discovery, 2)
	if err != nil || final.Operation != nil || len(final.Conflicts) != 0 {
		t.Fatalf("completed revert snapshot = %#v, err=%v", final, err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "base\n" {
		t.Fatalf("reverted contents = %q, err=%v", contents, err)
	}
}

func TestRevertConflictAbortAfterFreshRunnerParityScenario(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runner := git.NewRunner(root)
	for _, args := range [][]string{
		{"init", "-b", "main", "--", root},
		{"config", "user.name", "gitwatch-revert-abort-parity"},
		{"config", "user.email", "gitwatch-revert-abort-parity@example.com"},
		{"config", "commit.gpgsign", "false"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(root, "revert-abort.txt")
	commit := func(content, message string) string {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("revert-abort.txt")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		result, err := runner.Run(ctx, "rev-parse", "HEAD")
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(result.Stdout))
	}

	commit("base\n", "base")
	first := commit("first\n", "first change")
	commit("second\n", "second change")
	discovery, err := git.Discover(ctx, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, revertErr := runner.Revert(ctx, git.RevertRequest{Commits: []string{first}}); revertErr == nil {
		t.Fatal("revert unexpectedly completed without conflict")
	}

	// Reconstruct the process boundary as a restarted gitwatch instance would.
	restarted := git.NewRunner(root)
	operation, err := git.DetectOperationState(ctx, discovery, 7)
	if err != nil || !operation.Found || operation.State.Kind() != sequencer.KindRevert {
		t.Fatalf("fresh-runner revert operation = %#v, err=%v", operation, err)
	}
	if _, err := restarted.OperationLifecycle(ctx, sequencer.KindRevert, "abort"); err != nil {
		t.Fatal(err)
	}
	final, err := git.Snapshot(ctx, discovery, 8)
	if err != nil || final.Operation != nil || len(final.Conflicts) != 0 {
		t.Fatalf("aborted revert snapshot = %#v, err=%v", final, err)
	}
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != "second\n" {
		t.Fatalf("aborted revert contents = %q, err=%v", contents, err)
	}
}
