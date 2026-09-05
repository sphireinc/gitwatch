package git

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// RevertRequest describes one explicitly ordered revert operation. Commits
// are passed to Git in this order; callers should show that order before
// starting because reversing it can change the result.
type RevertRequest struct {
	Commits  []string
	Mainline int
}

// Revert starts a single- or multi-commit revert through Git's sequencer.
// A mainline parent is required by callers when the selected commit is a
// merge; Git remains authoritative for validating the commit itself.
func (r Runner) Revert(ctx context.Context, request RevertRequest) (OperationResult, error) {
	if err := request.Validate(); err != nil {
		return OperationResult{Name: "revert"}, err
	}
	args := []string{"revert", "--no-edit"}
	if request.Mainline > 0 {
		args = append(args, "-m", fmt.Sprintf("%d", request.Mainline))
	}
	args = append(args, request.Commits...)
	result, err := r.Run(ctx, args...)
	return OperationResult{Name: "revert", Result: result}, err
}

// Validate checks the command shape before any repository mutation occurs.
func (r RevertRequest) Validate() error {
	if len(r.Commits) == 0 {
		return errors.New("revert requires at least one commit")
	}
	if r.Mainline < 0 {
		return errors.New("revert mainline parent must be positive")
	}
	for _, commit := range r.Commits {
		value := strings.TrimSpace(commit)
		if value == "" || strings.HasPrefix(value, "-") || strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf("invalid revert commit %q", commit)
		}
	}
	return nil
}
