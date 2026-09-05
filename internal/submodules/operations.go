package submodules

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

var (
	ErrInvalidSubmodulePath = errors.New("invalid submodule path")
	ErrConfirmationRequired = errors.New("exact submodule path confirmation is required")
	ErrInvalidSubmoduleURL  = errors.New("invalid submodule URL")
)

// Request scopes a lifecycle operation to one repository and one path.
type Request struct {
	Repository string
	Path       string
}

// AddRequest describes a new submodule. Recursive behavior is intentionally
// not part of this operation; callers must issue separate explicit actions.
type AddRequest struct {
	Repository string
	Path       string
	URL        string
}

// RemoveRequest requires the UI to repeat the exact path shown to the user.
type RemoveRequest struct {
	Request
	ConfirmedPath string
}

// Outcome contains bounded command evidence. Sensitive URL text is redacted
// from the retained result and error before it can reach a journal or UI.
type Outcome struct {
	Action string
	Path   string
	Result git.Result
	Err    error
}

// CommandRunner is the narrow typed command boundary needed by lifecycle
// operations. git.Runner is the production implementation; the interface
// keeps bounded orchestration independently testable without shell fakes.
type CommandRunner interface {
	Run(context.Context, ...string) (git.Result, error)
}

func Initialize(ctx context.Context, runner CommandRunner, request Request) Outcome {
	return run(ctx, runner, "initialize", request, "submodule", "update", "--init", "--", request.Path)
}

func Update(ctx context.Context, runner CommandRunner, request Request) Outcome {
	return run(ctx, runner, "update", request, "submodule", "update", "--", request.Path)
}

func Sync(ctx context.Context, runner CommandRunner, request Request) Outcome {
	return run(ctx, runner, "sync", request, "submodule", "sync", "--", request.Path)
}

func Deinit(ctx context.Context, runner CommandRunner, request RemoveRequest) Outcome {
	if err := validateRemoval(request); err != nil {
		return Outcome{Action: "deinit", Path: request.Path, Err: err}
	}
	return run(ctx, runner, "deinit", request.Request, "submodule", "deinit", "--", request.Path)
}

func Remove(ctx context.Context, runner CommandRunner, request RemoveRequest) Outcome {
	if err := validateRemoval(request); err != nil {
		return Outcome{Action: "remove", Path: request.Path, Err: err}
	}
	return run(ctx, runner, "remove", request.Request, "rm", "--", request.Path)
}

func Add(ctx context.Context, runner CommandRunner, request AddRequest) Outcome {
	if err := validateRepositoryAndPath(request.Repository, request.Path); err != nil {
		return Outcome{Action: "add", Path: request.Path, Err: err}
	}
	if strings.TrimSpace(request.URL) == "" || strings.ContainsAny(request.URL, "\x00\r\n") {
		return Outcome{Action: "add", Path: request.Path, Err: ErrInvalidSubmoduleURL}
	}
	outcome := run(ctx, runner, "add", Request{Repository: request.Repository, Path: request.Path}, "submodule", "add", request.URL, request.Path)
	return sanitizeOutcome(outcome, request.URL)
}

func run(ctx context.Context, runner CommandRunner, action string, request Request, args ...string) Outcome {
	if err := validateRepositoryAndPath(request.Repository, request.Path); err != nil {
		return Outcome{Action: action, Path: request.Path, Err: err}
	}
	result, err := runner.Run(ctx, args...)
	outcome := Outcome{Action: action, Path: request.Path, Result: result, Err: err}
	return sanitizeOutcome(outcome)
}

func validateRemoval(request RemoveRequest) error {
	if err := validateRepositoryAndPath(request.Repository, request.Path); err != nil {
		return err
	}
	if request.ConfirmedPath != request.Path {
		return fmt.Errorf("%w: expected %q", ErrConfirmationRequired, request.Path)
	}
	return nil
}

func validateRepositoryAndPath(repository, path string) error {
	if strings.TrimSpace(repository) == "" || path == "" || strings.ContainsAny(path, "\x00\r\n") {
		return ErrInvalidSubmodulePath
	}
	if strings.HasPrefix(path, "/") || path == "." || path == ".." || strings.HasPrefix(path, "../") || strings.Contains(path, "/../") {
		return ErrInvalidSubmodulePath
	}
	return nil
}

func sanitizeOutcome(outcome Outcome, secrets ...string) Outcome {
	for _, secret := range secrets {
		redacted := RedactURL(secret)
		outcome.Result.Args = replaceStrings(outcome.Result.Args, secret, redacted)
		outcome.Result.Stdout = []byte(strings.ReplaceAll(string(outcome.Result.Stdout), secret, redacted))
		outcome.Result.Stderr = []byte(strings.ReplaceAll(string(outcome.Result.Stderr), secret, redacted))
		if outcome.Err != nil {
			outcome.Err = errors.New(strings.ReplaceAll(outcome.Err.Error(), secret, redacted))
		}
	}
	return outcome
}

func replaceStrings(values []string, old, replacement string) []string {
	copyValues := append([]string(nil), values...)
	for i := range copyValues {
		copyValues[i] = strings.ReplaceAll(copyValues[i], old, replacement)
	}
	return copyValues
}
