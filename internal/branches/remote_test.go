package branches

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type remoteCaptureRunner struct{ args []string }

func (r *remoteCaptureRunner) Run(_ context.Context, args ...string) (git.Result, error) {
	r.args = append([]string(nil), args...)
	return git.Result{Args: r.args}, nil
}

func TestRemoteBranchActionsUseQualifiedRefs(t *testing.T) {
	var runner remoteCaptureRunner
	if _, err := CheckoutRemote(context.Background(), &runner, "origin", "feature/x", "feature-x"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"switch", "--track", "--create", "feature-x", "origin/feature/x"}; !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("tracking checkout argv = %#v, want %#v", runner.args, want)
	}
	if _, err := CheckoutRemoteDetached(context.Background(), &runner, "backup", "feature/x"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"switch", "--detach", "--", "backup/feature/x"}; !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("detached checkout argv = %#v, want %#v", runner.args, want)
	}
	if _, err := DeleteRemote(context.Background(), &runner, "backup", "feature/x", "backup/feature/x"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"push", "backup", ":refs/heads/feature/x"}; !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("remote delete argv = %#v, want %#v", runner.args, want)
	}
}

func TestDeleteRemoteRequiresFullQualifiedConfirmation(t *testing.T) {
	var runner remoteCaptureRunner
	if _, err := DeleteRemote(context.Background(), &runner, "origin", "main", "main"); !errors.Is(err, ErrRemoteConfirmation) {
		t.Fatalf("confirmation error = %v", err)
	}
	if _, err := DeleteRemote(context.Background(), &runner, "origin/other", "main", "origin/other/main"); !errors.Is(err, ErrInvalidRemoteRef) {
		t.Fatalf("remote validation error = %v", err)
	}
}
