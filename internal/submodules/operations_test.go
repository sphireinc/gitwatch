package submodules

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestLifecycleValidationRejectsUnsafePathsAndRequiresExactRemovalConfirmation(t *testing.T) {
	invalid := Initialize(context.Background(), git.Runner{}, Request{Repository: "/repo", Path: "../outside"})
	if !errors.Is(invalid.Err, ErrInvalidSubmodulePath) {
		t.Fatalf("invalid path error = %v", invalid.Err)
	}
	missing := Remove(context.Background(), git.Runner{}, RemoveRequest{Request: Request{Repository: "/repo", Path: "nested path"}, ConfirmedPath: "nested"})
	if !errors.Is(missing.Err, ErrConfirmationRequired) {
		t.Fatalf("missing confirmation error = %v", missing.Err)
	}
	if err := validateRepositoryAndPath("/repo", "-leading-hyphen"); err != nil {
		t.Fatalf("argv-safe path rejected = %v", err)
	}
}

func TestAddValidationRejectsControlCharactersAndSanitizesURLResult(t *testing.T) {
	invalid := Add(context.Background(), git.Runner{}, AddRequest{Repository: "/repo", Path: "module", URL: "https://example.test/\nsecret"})
	if !errors.Is(invalid.Err, ErrInvalidSubmoduleURL) {
		t.Fatalf("invalid URL error = %v", invalid.Err)
	}
	secret := "https://alice:secret@example.test/repo"
	outcome := sanitizeOutcome(Outcome{
		Result: git.Result{Args: []string{"submodule", "add", secret, "module"}, Stdout: []byte(secret), Stderr: []byte(secret)},
		Err:    errors.New("failed " + secret),
	}, secret)
	redacted := RedactURL(secret)
	for _, value := range outcome.Result.Args {
		if value == secret {
			t.Fatalf("secret remained in args: %q", value)
		}
	}
	if string(outcome.Result.Stdout) != redacted || string(outcome.Result.Stderr) != redacted || outcome.Err.Error() != "failed "+redacted {
		t.Fatalf("sanitized outcome = %+v err=%v", outcome.Result, outcome.Err)
	}
}

func TestLifecycleOperationsUseRealLocalRepositoryAndExactRemoval(t *testing.T) {
	ctx := context.Background()
	child := t.TempDir()
	childRunner := git.NewRunner(child)
	gitMustRun(t, ctx, childRunner, "init", "-b", "main", "--", child)
	if err := os.WriteFile(filepath.Join(child, "README"), []byte("child\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRun(t, ctx, childRunner, "add", "--", "README")
	gitMustRun(t, ctx, childRunner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "child")

	parent := t.TempDir()
	runner := git.NewRunner(parent)
	runner.Env = []string{"GIT_ALLOW_PROTOCOL=file"}
	gitMustRun(t, ctx, runner, "init", "-b", "main", "--", parent)
	added := Add(ctx, runner, AddRequest{Repository: parent, Path: "nested path", URL: "file://" + child})
	if added.Err != nil {
		t.Fatalf("add outcome = %+v", added)
	}
	gitMustRun(t, ctx, runner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "add-submodule")
	initialized := Initialize(ctx, runner, Request{Repository: parent, Path: "nested path"})
	if initialized.Err != nil {
		t.Fatalf("initialize outcome = %+v", initialized)
	}
	synced := Sync(ctx, runner, Request{Repository: parent, Path: "nested path"})
	if synced.Err != nil {
		t.Fatalf("sync outcome = %+v", synced)
	}
	deinitialized := Deinit(ctx, runner, RemoveRequest{Request: Request{Repository: parent, Path: "nested path"}, ConfirmedPath: "nested path"})
	if deinitialized.Err != nil {
		t.Fatalf("deinit outcome = %+v", deinitialized)
	}
	removed := Remove(ctx, runner, RemoveRequest{Request: Request{Repository: parent, Path: "nested path"}, ConfirmedPath: "nested path"})
	if removed.Err != nil {
		t.Fatalf("remove outcome = %+v", removed)
	}
	if _, err := os.Stat(filepath.Join(parent, "nested path")); !os.IsNotExist(err) {
		t.Fatalf("removed submodule path still exists: err=%v", err)
	}
}
