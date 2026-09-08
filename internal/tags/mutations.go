package tags

import (
	"context"
	"errors"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
)

var (
	ErrInvalidTarget        = errors.New("invalid tag target")
	ErrConfirmationRequired = errors.New("exact tag name confirmation is required")
	ErrMessageRequired      = errors.New("annotated tag message is required")
)

// CreateKind controls whether a tag is lightweight, annotated, or signed.
type CreateKind string

const (
	CreateLightweight CreateKind = "lightweight"
	CreateAnnotated   CreateKind = "annotated"
	CreateSigned      CreateKind = "signed"
)

// CreateRequest describes one explicit local tag creation.
type CreateRequest struct {
	Name    string
	Target  string
	Message string
	Kind    CreateKind
}

// Create creates a local tag using typed argv arguments. Signing is delegated
// to Git's configured signing capability; this package never guesses a key.
func Create(ctx context.Context, runner VerifyRunner, request CreateRequest) (git.Result, error) {
	if !validName(request.Name) || !validTarget(request.Target) {
		return git.Result{}, ErrInvalidTarget
	}
	args := []string{"tag"}
	switch request.Kind {
	case CreateLightweight:
		args = append(args, request.Name, request.Target)
	case CreateAnnotated, CreateSigned:
		if strings.TrimSpace(request.Message) == "" || strings.ContainsAny(request.Message, "\x00\r\n") {
			return git.Result{}, ErrMessageRequired
		}
		if request.Kind == CreateAnnotated {
			args = append(args, "-a")
		} else {
			args = append(args, "-s")
		}
		args = append(args, "-m", request.Message, request.Name, request.Target)
	default:
		return git.Result{}, ErrInvalidTarget
	}
	return runner.Run(ctx, args...)
}

// Delete removes one local tag only after the UI repeats the exact name.
func Delete(ctx context.Context, runner VerifyRunner, name, confirmedName string) (git.Result, error) {
	if !validName(name) {
		return git.Result{}, ErrInvalidTarget
	}
	if name != confirmedName {
		return git.Result{}, ErrConfirmationRequired
	}
	return runner.Run(ctx, "tag", "-d", name)
}

func validTarget(target string) bool {
	target = strings.TrimSpace(target)
	return target != "" && !strings.HasPrefix(target, "-") && !strings.ContainsAny(target, "\x00\r\n")
}
