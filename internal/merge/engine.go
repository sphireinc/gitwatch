// Package merge provides a typed, guarded adapter for Git merge operations.
package merge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

// Strategy is an explicit merge strategy.
type Strategy uint8

const (
	Regular Strategy = iota
	FastForwardOnly
	NoFastForward
	Squash
)

func (s Strategy) String() string {
	switch s {
	case FastForwardOnly:
		return "ff-only"
	case NoFastForward:
		return "no-ff"
	case Squash:
		return "squash"
	default:
		return "regular"
	}
}

var (
	ErrInvalidSource   = errors.New("merge source ref is invalid")
	ErrDirtyWorktree   = errors.New("merge requires a clean worktree; stash explicitly first")
	ErrActiveOperation = errors.New("another Git operation is already active")
)

// Request describes a merge into the currently checked-out branch.
type Request struct {
	Repository string
	Generation uint64
	Source     string
	Strategy   Strategy
	Message    string
}

// Outcome retains Git's result and authoritative operation state.
type Outcome struct {
	Result git.Result
	Err    error
	Paused bool
	State  *sequencer.State
}

// Engine performs guarded merges in one repository.
type Engine struct {
	Runner     git.Runner
	Discovery  git.Discovery
	Repository string
	Generation uint64
}

// Execute performs a merge after authoritative dirty-worktree and operation
// preflight. It never stashes or resets on the caller's behalf.
func (e Engine) Execute(ctx context.Context, request Request) Outcome {
	if request.Repository == "" || request.Repository != e.Repository {
		return Outcome{Err: errors.New("merge request belongs to a different repository")}
	}
	if !validSource(request.Source) {
		return Outcome{Err: ErrInvalidSource}
	}
	if request.Strategy > Squash {
		return Outcome{Err: fmt.Errorf("unsupported merge strategy %d", request.Strategy)}
	}
	discovery := e.Discovery
	if discovery.Root == "" {
		var err error
		discovery, err = git.Discover(ctx, e.Runner.Dir)
		if err != nil {
			return Outcome{Err: err}
		}
	}
	operation, err := git.DetectOperationState(ctx, discovery, e.Generation)
	if err != nil {
		return Outcome{Err: err}
	}
	if operation.Found {
		return Outcome{Err: ErrActiveOperation, Paused: true, State: statePtr(operation.State)}
	}
	snapshot, err := git.Snapshot(ctx, discovery, e.Generation)
	if err != nil {
		return Outcome{Err: err}
	}
	if len(snapshot.Entries) > 0 {
		return Outcome{Err: ErrDirtyWorktree}
	}
	args := []string{"merge"}
	switch request.Strategy {
	case FastForwardOnly:
		args = append(args, "--ff-only")
	case NoFastForward:
		args = append(args, "--no-ff")
	case Squash:
		args = append(args, "--squash")
	case Regular:
	default:
		return Outcome{Err: fmt.Errorf("unsupported merge strategy %d", request.Strategy)}
	}
	if request.Message != "" {
		if strings.ContainsAny(request.Message, "\r\n") {
			return Outcome{Err: errors.New("merge message must be one line")}
		}
		args = append(args, "-m", request.Message)
	}
	args = append(args, request.Source)
	result, commandErr := e.Runner.Run(ctx, args...)
	outcome := Outcome{Result: result, Err: commandErr}
	updated, discoverErr := git.Discover(ctx, e.Runner.Dir)
	if discoverErr != nil {
		return outcome
	}
	observed, detectErr := git.DetectOperationState(ctx, updated, e.Generation)
	if detectErr == nil && observed.Found && observed.State.Kind() == sequencer.KindMerge {
		outcome.Paused, outcome.State = true, statePtr(observed.State)
	}
	return outcome
}

func statePtr(state sequencer.State) *sequencer.State { return &state }
func validSource(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00 \t")
}
