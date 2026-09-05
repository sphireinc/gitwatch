package submodules

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sphireinc/git-watch/internal/git"
)

const (
	defaultBulkWorkers = 4
	maxBulkWorkers     = 8
	defaultBulkModules = 256
)

// BulkAction identifies a non-destructive bulk lifecycle operation.
type BulkAction string

const (
	BulkInitialize BulkAction = "initialize"
	BulkUpdate     BulkAction = "update"
	BulkSync       BulkAction = "sync"
)

// ItemState describes one observable bulk-operation state.
type ItemState string

const (
	ItemQueued    ItemState = "queued"
	ItemRunning   ItemState = "running"
	ItemSucceeded ItemState = "succeeded"
	ItemFailed    ItemState = "failed"
	ItemSkipped   ItemState = "skipped"
	ItemCancelled ItemState = "cancelled"
)

// BulkRequest is explicitly scoped to the selected paths supplied by the UI.
// An empty Paths slice is rejected; callers must make an all-submodules choice
// explicit by passing the loaded module paths.
type BulkRequest struct {
	Repository string
	Paths      []string
	Action     BulkAction
	Workers    int
	MaxModules int
}

// BulkItem records one module's lifecycle result in input order.
type BulkItem struct {
	Path    string
	State   ItemState
	Outcome Outcome
}

// BulkOutcome retains every per-module result instead of failing fast.
type BulkOutcome struct {
	Repository string
	Action     BulkAction
	Items      []BulkItem
	Cancelled  bool
}

// Bulk executes selected initialize/update/sync operations with a hard worker
// bound. Each Git child inherits ctx, and a failed module never prevents other
// queued modules from running.
func Bulk(ctx context.Context, runner CommandRunner, request BulkRequest) BulkOutcome {
	outcome := BulkOutcome{Repository: request.Repository, Action: request.Action}
	if ctx == nil {
		ctx = context.Background()
	}
	maxModules := request.MaxModules
	if maxModules <= 0 {
		maxModules = defaultBulkModules
	}
	paths := deduplicatePaths(request.Paths)
	outcome.Items = make([]BulkItem, len(paths))
	for i, path := range paths {
		outcome.Items[i] = BulkItem{Path: path, State: ItemQueued}
		if i >= maxModules {
			outcome.Items[i].State = ItemSkipped
			outcome.Items[i].Outcome = Outcome{Action: string(request.Action), Path: path, Err: ErrBulkLimit}
			continue
		}
		if err := validateRepositoryAndPath(request.Repository, path); err != nil {
			outcome.Items[i].State = ItemFailed
			outcome.Items[i].Outcome = Outcome{Action: string(request.Action), Path: path, Err: err}
		}
	}
	if request.Repository == "" {
		for i := range outcome.Items {
			if outcome.Items[i].State == ItemQueued {
				outcome.Items[i].State = ItemFailed
				outcome.Items[i].Outcome.Err = ErrInvalidSubmodulePath
			}
		}
		return outcome
	}
	if request.Action != BulkInitialize && request.Action != BulkUpdate && request.Action != BulkSync {
		for i := range outcome.Items {
			if outcome.Items[i].State == ItemQueued {
				outcome.Items[i].State = ItemFailed
				outcome.Items[i].Outcome.Err = fmt.Errorf("unsupported bulk submodule action %q", request.Action)
			}
		}
		return outcome
	}
	workers := request.Workers
	if workers <= 0 {
		workers = defaultBulkWorkers
	}
	if workers > maxBulkWorkers {
		workers = maxBulkWorkers
	}
	queued := make(chan int, len(outcome.Items))
	for i := range outcome.Items {
		if outcome.Items[i].State == ItemQueued {
			queued <- i
		}
	}
	close(queued)
	var group sync.WaitGroup
	if workers > len(queued) {
		workers = len(queued)
	}
	for n := 0; n < workers; n++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range queued {
				if ctx.Err() != nil {
					outcome.Items[index].State = ItemCancelled
					outcome.Items[index].Outcome.Err = ctx.Err()
					continue
				}
				outcome.Items[index].State = ItemRunning
				outcome.Items[index].Outcome = runBulkAction(ctx, runner, request.Action, Request{Repository: request.Repository, Path: outcome.Items[index].Path})
				if outcome.Items[index].Outcome.Err != nil {
					if errors.Is(outcome.Items[index].Outcome.Err, context.Canceled) || errors.Is(outcome.Items[index].Outcome.Err, git.ErrCancelled) {
						outcome.Items[index].State = ItemCancelled
					} else {
						outcome.Items[index].State = ItemFailed
					}
				} else {
					outcome.Items[index].State = ItemSucceeded
				}
			}
		}()
	}
	group.Wait()
	if ctx.Err() != nil {
		outcome.Cancelled = true
	}
	return outcome
}

var ErrBulkLimit = errors.New("bulk submodule module limit exceeded")

func runBulkAction(ctx context.Context, runner CommandRunner, action BulkAction, request Request) Outcome {
	switch action {
	case BulkInitialize:
		return Initialize(ctx, runner, request)
	case BulkUpdate:
		return Update(ctx, runner, request)
	case BulkSync:
		return Sync(ctx, runner, request)
	default:
		return Outcome{Action: string(action), Path: request.Path, Err: fmt.Errorf("unsupported bulk submodule action %q", action)}
	}
}

func deduplicatePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	return unique
}
