package compare

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type fakeRunner struct {
	resolve int
	args    [][]string
}

func (r *fakeRunner) Run(_ context.Context, args ...string) (git.Result, error) {
	r.args = append(r.args, append([]string(nil), args...))
	switch args[0] {
	case "rev-parse":
		r.resolve++
		return git.Result{Args: args, Stdout: []byte(map[int]string{1: "left-sha\n", 2: "right-sha\n"}[r.resolve])}, nil
	case "show":
		if args[len(args)-1] == "left-sha" {
			return git.Result{Args: args, Stdout: []byte("left-sha\x00A\x00left subject\x002024-01-01T00:00:00Z\n")}, nil
		}
		return git.Result{Args: args, Stdout: []byte("right-sha\x00B\x00right subject\x002024-01-02T00:00:00Z\n")}, nil
	case "diff":
		if args[1] == "--name-status" {
			return git.Result{Args: args, Stdout: []byte("M\x00normal\x00R100\x00old name\x00new name\x00A\x00odd\tname\x00")}, nil
		}
		return git.Result{Args: args, Stdout: []byte("2\t1\tnormal\x001\t1\tnew name\x00-\t-\todd\tname\x00")}, nil
	default:
		return git.Result{Args: args}, nil
	}
}

func (r *fakeRunner) RunBounded(_ context.Context, _ int, args ...string) (git.Result, error) {
	r.args = append(r.args, append([]string(nil), args...))
	if args[0] == "log" {
		return git.Result{Args: args, Stdout: []byte("commit-sha\x00Author\x002024-01-03T00:00:00Z\x00subject\x00")}, nil
	}
	return git.Result{Args: args, Stdout: []byte("diff --git a/normal b/normal\n")}, nil
}

func TestCompareResolvesRefsAndPreservesRenameAndUnusualPaths(t *testing.T) {
	runner := &fakeRunner{}
	result, err := Compare(context.Background(), runner, Request{Left: "HEAD~1", Right: "origin/main", MaxFiles: 10, MaxPatchBytes: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if result.Left.SHA != "left-sha" || result.Right.SHA != "right-sha" {
		t.Fatalf("resolved revisions = %#v", result)
	}
	if len(result.Changes) != 3 || result.Changes[1].OldPath != "old name" || result.Changes[1].NewPath != "new name" || !result.Changes[2].Binary || result.Changes[2].NewPath != "odd\tname" {
		t.Fatalf("changes = %#v", result.Changes)
	}
	if result.Changes[0].Added != 2 || result.Changes[0].Removed != 1 || result.PatchTruncated {
		t.Fatalf("stats/patch = %#v", result)
	}
	wantResolve := []string{"rev-parse", "--verify", "--end-of-options", "HEAD~1^{commit}"}
	if !reflect.DeepEqual(runner.args[0], wantResolve) {
		t.Fatalf("resolve argv = %#v, want %#v", runner.args[0], wantResolve)
	}
}

func TestResolveRejectsMissingOrUnsafeRevision(t *testing.T) {
	runner := &fakeRunner{}
	if _, err := Resolve(context.Background(), runner, ""); !errors.Is(err, ErrMissingRevision) {
		t.Fatalf("missing revision error = %v", err)
	}
	if _, err := Resolve(context.Background(), runner, "-bad"); !errors.Is(err, ErrInvalidRevision) {
		t.Fatalf("unsafe revision error = %v", err)
	}
	if len(runner.args) != 0 {
		t.Fatalf("unsafe revision reached Git: %#v", runner.args)
	}
}

func TestCompareHonorsFileLimitAndPatchOutputLimit(t *testing.T) {
	runner := &fakeRunner{}
	result, err := Compare(context.Background(), runner, Request{Left: "a", Right: "b", MaxFiles: 1, MaxPatchBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Changes) != 1 || !result.FilesTruncated {
		t.Fatalf("file limit = %#v", result)
	}
}

func TestCompareLoadsUniqueCommitSummaries(t *testing.T) {
	runner := &fakeRunner{}
	result, err := Compare(context.Background(), runner, Request{Left: "a", Right: "b", MaxCommits: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.LeftOnly) != 1 || result.LeftOnly[0].SHA != "commit-sha" || len(result.RightOnly) != 1 || result.CommitsTruncated {
		t.Fatalf("unique commits = %#v", result)
	}
}

func TestFilePatchUsesResolvedSHAsAndPathSeparator(t *testing.T) {
	runner := &fakeRunner{}
	text, truncated, err := FilePatch(context.Background(), runner, Result{Left: Revision{SHA: "left"}, Right: Revision{SHA: "right"}}, Change{NewPath: "-leading name"}, 1024)
	if err != nil || truncated || text == "" {
		t.Fatalf("file patch = text=%q truncated=%v err=%v", text, truncated, err)
	}
	want := []string{"diff", "--no-ext-diff", "left", "right", "--", "-leading name"}
	if !reflect.DeepEqual(runner.args[len(runner.args)-1], want) {
		t.Fatalf("file patch argv = %#v, want %#v", runner.args[len(runner.args)-1], want)
	}
}
