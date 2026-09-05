package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
