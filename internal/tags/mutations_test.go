package tags

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type captureRunner struct {
	args []string
}

func (r *captureRunner) Run(_ context.Context, args ...string) (git.Result, error) {
	r.args = append([]string(nil), args...)
	return git.Result{Args: r.args}, nil
}

func TestCreateBuildsExplicitLightweightAnnotatedAndSignedArgv(t *testing.T) {
	for _, test := range []struct {
		kind CreateKind
		want []string
	}{
		{CreateLightweight, []string{"tag", "v1", "HEAD"}},
		{CreateAnnotated, []string{"tag", "-a", "-m", "release", "v1", "HEAD"}},
		{CreateSigned, []string{"tag", "-s", "-m", "release", "v1", "HEAD"}},
	} {
		var runner captureRunner
		if _, err := Create(context.Background(), &runner, CreateRequest{Name: "v1", Target: "HEAD", Message: "release", Kind: test.kind}); err != nil {
			t.Fatalf("create %s: %v", test.kind, err)
		}
		if got := runner.args; !reflect.DeepEqual(got, test.want) {
			t.Fatalf("create %s argv = %#v, want %#v", test.kind, got, test.want)
		}
	}
}

func TestDeleteRequiresExactNameConfirmation(t *testing.T) {
	var runner captureRunner
	if _, err := Delete(context.Background(), &runner, "v1", "v2"); !errors.Is(err, ErrConfirmationRequired) {
		t.Fatalf("delete confirmation error = %v", err)
	}
	if _, err := Delete(context.Background(), &runner, "v1", "v1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runner.args, []string{"tag", "-d", "v1"}) {
		t.Fatalf("delete argv = %#v", runner.args)
	}
}

func TestCreateRejectsUnsafeOrIncompleteRequests(t *testing.T) {
	var runner captureRunner
	for _, request := range []CreateRequest{
		{Name: "-bad", Target: "HEAD", Kind: CreateLightweight},
		{Name: "v1", Target: "-bad", Kind: CreateLightweight},
		{Name: "v1", Target: "HEAD", Kind: CreateAnnotated},
		{Name: "v1", Target: "HEAD", Message: "signed", Kind: CreateKind("unknown")},
	} {
		if _, err := Create(context.Background(), &runner, request); err == nil {
			t.Fatalf("accepted invalid request: %+v", request)
		}
	}
}
