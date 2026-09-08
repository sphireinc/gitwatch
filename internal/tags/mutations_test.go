package tags

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

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

func TestSignedCreateFailsSafelyWithoutSigningKey(t *testing.T) {
	repository := t.TempDir()
	gnupgHome := t.TempDir()
	runner := git.NewRunner(repository)
	runner.Env = []string{"GNUPGHOME=" + gnupgHome, "GPG_TTY=/dev/null"}
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "gitwatch test"},
		{"config", "user.email", "gitwatch@example.test"},
		{"-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "initial"},
	} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	if err := os.Chmod(gnupgHome, 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := Create(ctx, runner, CreateRequest{Name: "v-no-key", Target: "HEAD", Message: "release", Kind: CreateSigned})
	if err == nil {
		t.Fatal("signed tag unexpectedly succeeded without a signing key")
	}
	if result.ExitCode == 0 {
		t.Fatalf("signed tag failure returned success result: %#v", result)
	}
}
