// Package redo contains conservative replay policies for gitwatch undo records.
package redo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/undo"
)

var (
	ErrUnsupportedOperation = errors.New("operation has no safe automatic redo policy")
	ErrRepositoryMismatch   = errors.New("redo record belongs to another repository")
	ErrDiverged             = errors.New("repository diverged from the recorded post-undo state")
	ErrActiveOperation      = errors.New("repository has an active Git operation")
	ErrInvalidHead          = errors.New("redo record contains an invalid object ID")
)

// Request contains the evidence captured by a successful semantic undo.
type Request struct {
	Repository       string
	Kind             string
	Ref              string
	OldHead          string
	NewHead          string
	PostSnapshotHash string
	Discovery        git.Discovery
	Generation       uint64
}

type Outcome struct {
	Result   git.Result
	Snapshot repo.Snapshot
	Err      error
}

// Plan permits replay only for gitwatch's own soft undo record. OldHead is the
// commit that was undone and NewHead is the expected current HEAD.
func Plan(request Request, snapshot repo.Snapshot) ([]string, error) {
	if request.Repository == "" || request.Repository != snapshot.Root {
		return nil, ErrRepositoryMismatch
	}
	if request.Kind != "undo commit" {
		return nil, ErrUnsupportedOperation
	}
	if snapshot.Operation != nil {
		return nil, ErrActiveOperation
	}
	if !validObjectID(request.OldHead) || !validObjectID(request.NewHead) || request.OldHead == request.NewHead {
		return nil, ErrInvalidHead
	}
	if snapshot.Branch.OID != request.NewHead || request.Ref == "" || snapshot.Branch.Name != request.Ref {
		return nil, ErrDiverged
	}
	if request.PostSnapshotHash == "" || undo.SnapshotFingerprint(snapshot) != request.PostSnapshotHash {
		return nil, ErrDiverged
	}
	return []string{"reset", "--soft", request.OldHead}, nil
}

// Execute re-reads status before replay and refreshes it after Git completes.
func Execute(ctx context.Context, runner git.Runner, request Request) Outcome {
	if request.Discovery.Root == "" {
		return Outcome{Err: ErrRepositoryMismatch}
	}
	snapshot, err := git.Snapshot(ctx, request.Discovery, request.Generation)
	if err != nil {
		return Outcome{Err: fmt.Errorf("redo status: %w", err)}
	}
	args, err := Plan(request, snapshot)
	if err != nil {
		return Outcome{Snapshot: snapshot, Err: err}
	}
	result, err := runner.Run(ctx, args...)
	outcome := Outcome{Result: result, Snapshot: snapshot, Err: err}
	if err == nil {
		if refreshed, refreshErr := git.Snapshot(ctx, request.Discovery, request.Generation); refreshErr == nil {
			outcome.Snapshot = refreshed
		} else {
			outcome.Err = fmt.Errorf("redo refresh: %w", refreshErr)
		}
	}
	return outcome
}

func validObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	return strings.Trim(value, "0123456789abcdefABCDEF") == ""
}
