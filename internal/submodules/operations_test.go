package submodules

import (
	"context"
	"errors"
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
