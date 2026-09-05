// Package merge provides a typed, guarded adapter for Git merge operations.
package merge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/sequencer"
	"github.com/sphireinc/git-watch/internal/worktrees"
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
	ErrCurrentBranch   = errors.New("merge source is the current branch")
	ErrSourceOccupied  = errors.New("merge source branch is checked out in another worktree")
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
	// Snapshot is the authoritative post-command repository refresh. It is
	// populated even when Git reports a conflict or command failure, when the
	// refresh itself succeeds.
	Snapshot *repo.Snapshot
}

// Abort restores the state recorded by Git for a paused merge.
func (e Engine) Abort(ctx context.Context) Outcome {
	result, err := e.Runner.Run(ctx, "merge", "--abort")
	outcome := Outcome{Result: result, Err: err}
	discovery := e.Discovery
	if discovery.Root == "" {
		discovery, _ = git.Discover(ctx, e.Runner.Dir)
	}
	if discovery.Root != "" {
		refresh, refreshErr := git.Snapshot(ctx, discovery, e.Generation)
		if refreshErr == nil {
			outcome.Snapshot = &refresh
			if refresh.Operation != nil && refresh.Operation.Kind() == sequencer.KindMerge {
				outcome.Paused, outcome.State = true, refresh.Operation
			}
		} else if outcome.Err == nil {
			outcome.Err = refreshErr
		}
	}
	return outcome
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
	if request.Source == snapshot.Branch.Name {
		return Outcome{Err: ErrCurrentBranch}
	}
	worktreeEntries, err := worktrees.List(ctx, git.NewRunner(discovery.Root))
	if err != nil {
		return Outcome{Err: fmt.Errorf("inspect worktree occupancy: %w", err)}
	}
	for _, entry := range worktreeEntries {
		if strings.TrimPrefix(entry.Branch, "refs/heads/") == request.Source && entry.Path != discovery.Root {
			return Outcome{Err: fmt.Errorf("%w: %s", ErrSourceOccupied, entry.Path)}
		}
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
		if outcome.Err == nil {
			outcome.Err = discoverErr
		}
		return outcome
	}
	refreshed, refreshErr := git.Snapshot(ctx, updated, e.Generation)
	if refreshErr != nil {
		if outcome.Err == nil {
			outcome.Err = refreshErr
		}
		return outcome
	}
	outcome.Snapshot = &refreshed
	if refreshed.Operation != nil && refreshed.Operation.Kind() == sequencer.KindMerge {
		outcome.Paused, outcome.State = true, refreshed.Operation
	}
	return outcome
}

func statePtr(state sequencer.State) *sequencer.State { return &state }
func validSource(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00 \t")
}
