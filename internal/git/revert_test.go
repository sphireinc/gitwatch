package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/sequencer"
)

func TestRevertRequestValidatesOrderedCommits(t *testing.T) {
	for _, test := range []struct {
		name    string
		request RevertRequest
		want    string
	}{
		{"empty", RevertRequest{}, "revert requires at least one commit"},
		{"option", RevertRequest{Commits: []string{"--no-edit"}}, `invalid revert commit "--no-edit"`},
		{"negative mainline", RevertRequest{Commits: []string{"abc"}, Mainline: -1}, "revert mainline parent must be positive"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.request.Validate(); err == nil || err.Error() != test.want {
				t.Fatalf("validation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRunnerRevertRejectsInvalidRequestBeforeGit(t *testing.T) {
	_, err := (Runner{}).Revert(context.Background(), RevertRequest{Commits: []string{""}})
	if err == nil {
		t.Fatalf("invalid request error = %v", err)
	}
}

func TestRunnerRevertAppliesMultipleCommitsInRequestedOrder(t *testing.T) {
	runner, discovery := operationFixture(t)
	first := filepath.Join(discovery.Root, "first.txt")
	if err := os.WriteFile(first, []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "add", "--", "first.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "commit", "-m", "first"); err != nil {
		t.Fatal(err)
	}
	firstSHA := rev(t, runner, "HEAD")
	second := filepath.Join(discovery.Root, "second.txt")
	if err := os.WriteFile(second, []byte("second\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "add", "--", "second.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "commit", "-m", "second"); err != nil {
		t.Fatal(err)
	}
	secondSHA := rev(t, runner, "HEAD")

	result, err := runner.Revert(context.Background(), RevertRequest{Commits: []string{secondSHA, firstSHA}})
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := []string{"revert", "--no-edit", secondSHA, firstSHA}
	if len(result.Result.Args) != len(wantArgs) {
		t.Fatalf("revert args = %#v, want %#v", result.Result.Args, wantArgs)
	}
	for i := range wantArgs {
		if result.Result.Args[i] != wantArgs[i] {
			t.Fatalf("revert args = %#v, want %#v", result.Result.Args, wantArgs)
		}
	}
	if _, err := os.Stat(first); !os.IsNotExist(err) {
		t.Fatalf("first reverted file still exists: %v", err)
	}
	if _, err := os.Stat(second); !os.IsNotExist(err) {
		t.Fatalf("second reverted file still exists: %v", err)
	}
}

func TestRevertMiddleConflictRecoversProgressAndSkipsAfterRestart(t *testing.T) {
	ctx := context.Background()
	runner, discovery := operationFixture(t)
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
	commitFile(t, runner, discovery.Root, "second\n", "second")
	second := rev(t, runner, "HEAD")
	third := commitNamed("third.txt", "third\n", "third")
	commitFile(t, runner, discovery.Root, "diverged\n", "diverged")
	original := rev(t, runner, "HEAD")
	if _, err := runner.Revert(ctx, RevertRequest{Commits: []string{third, second, first}}); err == nil {
		t.Fatal("expected middle revert conflict")
	}
	t.Cleanup(func() { _, _ = runner.Run(context.Background(), "revert", "--abort") })
	restarted, err := Discover(ctx, discovery.Root)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DetectOperationState(ctx, restarted, 57)
	if err != nil {
		t.Fatal(err)
	}
	state := got.State
	details := state.Details().Revert
	if !got.Found || state.Kind() != sequencer.KindRevert || state.HeadBefore() != original || state.Completed() != 1 || state.Remaining() != 2 || details == nil || len(details.Commits) != 3 || len(details.Completed) != 1 || details.CurrentIndex != 1 {
		t.Fatalf("restarted middle-revert state = found=%v state=%#v details=%#v", got.Found, state, details)
	}
	if _, err := NewRunner(discovery.Root).OperationLifecycle(ctx, sequencer.KindRevert, "skip"); err != nil {
		t.Fatal(err)
	}
	finished, err := DetectOperationState(ctx, restarted, 58)
	if err != nil || finished.Found {
		t.Fatalf("revert remains active after skip: %#v, err=%v", finished, err)
	}
	for _, path := range []string{"first.txt", "third.txt"} {
		if _, err := os.Stat(filepath.Join(discovery.Root, path)); !os.IsNotExist(err) {
			t.Fatalf("reverted file %q remains: %v", path, err)
		}
	}
}
