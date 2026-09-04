// Package cherrypick provides a repository-scoped, typed adapter for Git's
// cherry-pick sequencer. It owns no UI state and never constructs shell text.
package cherrypick

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

var (
	ErrEmptySelection   = errors.New("cherry-pick selection is empty")
	ErrMainlineRequired = errors.New("merge commit requires explicit mainline parent")
	ErrInvalidMainline  = errors.New("cherry-pick mainline parent is invalid")
)

// Request describes one ordered, repository-scoped cherry-pick operation.
type Request struct {
	Repository string
	Generation uint64
	SHAs       []string
	Mainline   int
}

// Journal captures the bounded recovery information for one operation.
type Journal struct {
	Repository   string
	Generation   uint64
	OriginalHEAD string
	SHAs         []string
}

// Outcome retains Git's result and the authoritative sequencer observation.
type Outcome struct {
	Journal Journal
	Result  git.Result
	Err     error
	Paused  bool
	State   *sequencer.State
}

// Engine executes cherry-pick commands for one repository.
type Engine struct {
	Runner     git.Runner
	Discovery  git.Discovery
	Repository string
	Generation uint64
}

// Execute validates and runs one ordered cherry-pick operation.
func (e Engine) Execute(ctx context.Context, request Request) Outcome {
	journal, err := e.prepare(ctx, request)
	if err != nil {
		return Outcome{Journal: journal, Err: err}
	}
	args := []string{"cherry-pick"}
	if request.Mainline > 0 {
		args = append(args, "-m", strconv.Itoa(request.Mainline))
	}
	args = append(args, request.SHAs...)
	result, commandErr := e.Runner.Run(ctx, args...)
	return e.finish(ctx, journal, result, commandErr)
}

// Continue resumes an in-progress cherry-pick after conflict resolution.
func (e Engine) Continue(ctx context.Context) Outcome {
	return e.lifecycle(ctx, "--continue")
}

// Skip skips the current conflicted cherry-pick.
func (e Engine) Skip(ctx context.Context) Outcome { return e.lifecycle(ctx, "--skip") }

// Abort asks Git to restore the recorded original state.
func (e Engine) Abort(ctx context.Context) Outcome { return e.lifecycle(ctx, "--abort") }

func (e Engine) lifecycle(ctx context.Context, action string) Outcome {
	journal := Journal{Repository: e.Repository, Generation: e.Generation}
	result, commandErr := e.Runner.Run(ctx, "cherry-pick", action)
	return e.finish(ctx, journal, result, commandErr)
}

func (e Engine) prepare(ctx context.Context, request Request) (Journal, error) {
	if request.Repository == "" || request.Repository != e.Repository {
		return Journal{}, errors.New("cherry-pick selection belongs to a different repository")
	}
	if request.Generation != 0 && e.Generation != 0 && request.Generation != e.Generation {
		return Journal{}, errors.New("cherry-pick selection generation is stale")
	}
	if len(request.SHAs) == 0 {
		return Journal{}, ErrEmptySelection
	}
	seen := make(map[string]struct{}, len(request.SHAs))
	for _, sha := range request.SHAs {
		if !validSHA(sha) {
			return Journal{}, fmt.Errorf("invalid cherry-pick object name %q", sha)
		}
		if _, exists := seen[sha]; exists {
			return Journal{}, fmt.Errorf("duplicate cherry-pick object name %q", sha)
		}
		seen[sha] = struct{}{}
	}
	if request.Mainline < 0 {
		return Journal{}, ErrInvalidMainline
	}
	for _, sha := range request.SHAs {
		parents, err := e.parents(ctx, sha)
		if err != nil {
			return Journal{}, fmt.Errorf("resolve cherry-pick %s: %w", sha, err)
		}
		if len(parents) > 1 && request.Mainline == 0 {
			return Journal{}, fmt.Errorf("%s: %w", sha, ErrMainlineRequired)
		}
		if len(parents) > 1 && request.Mainline > len(parents) {
			return Journal{}, fmt.Errorf("%s: %w", sha, ErrInvalidMainline)
		}
	}
	head, err := e.Runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return Journal{}, fmt.Errorf("capture cherry-pick original HEAD: %w", err)
	}
	return Journal{Repository: e.Repository, Generation: e.Generation, OriginalHEAD: strings.TrimSpace(string(head.Stdout)), SHAs: append([]string(nil), request.SHAs...)}, nil
}

func (e Engine) parents(ctx context.Context, sha string) ([]string, error) {
	result, err := e.Runner.Run(ctx, "rev-list", "--parents", "-n", "1", sha)
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(result.Stdout))
	if len(fields) == 0 || fields[0] != sha {
		return nil, errors.New("Git returned an unexpected commit record")
	}
	return fields[1:], nil
}

func (e Engine) finish(ctx context.Context, journal Journal, result git.Result, commandErr error) Outcome {
	outcome := Outcome{Journal: journal, Result: result, Err: commandErr}
	discovery := e.Discovery
	if discovery.Root == "" {
		var err error
		discovery, err = git.Discover(ctx, e.Runner.Dir)
		if err != nil {
			return outcome
		}
	}
	observed, err := git.DetectOperationState(ctx, discovery, e.Generation)
	if err == nil && observed.Found && observed.State.Kind() == sequencer.KindCherryPick {
		outcome.Paused = true
		state := observed.State
		outcome.State = &state
	}
	return outcome
}

func validSHA(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00 \t")
}
