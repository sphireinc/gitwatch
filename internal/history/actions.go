package history

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jusanchez/gitwatch/internal/git"
)

var ErrMissingTarget = errors.New("history action requires an explicit target")

type Ref struct {
	Name string
	OID  string
	Kind string
}

type RevertConfirmation struct {
	SHA     string
	Subject string
}

func (c RevertConfirmation) Text() string {
	return fmt.Sprintf("Revert %s (%s)?", c.SHA, c.Subject)
}

func (c RevertConfirmation) Accept(input string) bool {
	return strings.TrimSpace(input) == c.SHA
}

func CheckoutCommit(ctx context.Context, runner git.Runner, sha string) (git.Result, error) {
	if !validTarget(sha) {
		return git.Result{}, ErrMissingTarget
	}
	return runner.Run(ctx, "switch", "--detach", "--", sha)
}

func CreateBranchAt(ctx context.Context, runner git.Runner, name, sha string) (git.Result, error) {
	if !validTarget(name) || !validTarget(sha) {
		return git.Result{}, ErrMissingTarget
	}
	return runner.Run(ctx, "switch", "--create", name, "--", sha)
}

func ListTags(ctx context.Context, runner git.Runner) ([]Ref, error) {
	result, err := runner.Run(ctx, "for-each-ref", "--format=%(refname:short)\x00%(objectname)", "refs/tags")
	if err != nil {
		return nil, err
	}
	return parseRefs(result.Stdout, "tag"), nil
}

func Revert(ctx context.Context, runner git.Runner, confirmation RevertConfirmation, input string) (git.Result, error) {
	if !validTarget(confirmation.SHA) || !confirmation.Accept(input) {
		return git.Result{}, ErrMissingTarget
	}
	return runner.Run(ctx, "revert", "--no-edit", confirmation.SHA)
}

func validTarget(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00")
}

func parseRefs(data []byte, kind string) []Ref {
	var refs []Ref
	for _, record := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		fields := strings.Split(record, "\x00")
		if len(fields) == 2 && fields[0] != "" {
			refs = append(refs, Ref{Name: fields[0], OID: fields[1], Kind: kind})
		}
	}
	return refs
}
