package branches

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type recoveryCaptureRunner struct{ args []string }

func (r *recoveryCaptureRunner) Run(_ context.Context, args ...string) (git.Result, error) {
	r.args = append([]string(nil), args...)
	return git.Result{Args: r.args}, nil
}

func TestFastForwardUsesQualifiedRefAndFFOnly(t *testing.T) {
	var runner recoveryCaptureRunner
	if _, err := FastForward(context.Background(), &runner, "origin/main"); err != nil {
		t.Fatal(err)
	}
	want := []string{"merge", "--ff-only", "--", "origin/main"}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("fast-forward argv = %#v, want %#v", runner.args, want)
	}
}

func TestResetModesPreserveExplicitSafeSemantics(t *testing.T) {
	for _, test := range []struct {
		mode ResetMode
		flag string
	}{
		{ResetSoft, "--soft"},
		{ResetMixed, "--mixed"},
	} {
		var runner recoveryCaptureRunner
		if _, err := Reset(context.Background(), &runner, test.mode, "HEAD~2"); err != nil {
			t.Fatal(err)
		}
		want := []string{"reset", test.flag, "--", "HEAD~2"}
		if !reflect.DeepEqual(runner.args, want) {
			t.Fatalf("reset argv = %#v, want %#v", runner.args, want)
		}
	}
}

func TestRecoveryRejectsUnsafeRefsAndHardResetMode(t *testing.T) {
	var runner recoveryCaptureRunner
	if _, err := FastForward(context.Background(), &runner, "-bad"); !errors.Is(err, ErrInvalidRecoveryRef) {
		t.Fatalf("fast-forward validation error = %v", err)
	}
	if _, err := Reset(context.Background(), &runner, ResetMode(99), "HEAD"); err == nil {
		t.Fatal("unsupported reset mode accepted")
	}
	if _, err := Reset(context.Background(), &runner, ResetSoft, "HEAD\nother"); !errors.Is(err, ErrInvalidRecoveryRef) {
		t.Fatalf("reset validation error = %v", err)
	}
}
