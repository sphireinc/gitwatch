package registry

import (
	"context"
	"sync"
	"time"

	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/repo"
)

type StatusResult struct {
	Repository Repository
	Snapshot   repo.Snapshot
	Error      error
	Skipped    bool
}

type Engine struct {
	Workers       int
	InactiveAfter time.Duration
	Discover      func(context.Context, string) (git.Discovery, error)
	Snapshot      func(context.Context, git.Discovery, uint64) (repo.Snapshot, error)
	mu            sync.Mutex
	cache         map[string]StatusResult
}

func NewEngine(workers int) *Engine {
	if workers < 1 {
		workers = 1
	}
	return &Engine{Workers: workers, InactiveAfter: 5 * time.Minute, Discover: git.Discover, Snapshot: git.Snapshot, cache: make(map[string]StatusResult)}
}

func (e *Engine) Refresh(ctx context.Context, repositories []Repository, activePath string) []StatusResult {
	workers := e.Workers
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan Repository)
	results := make(chan StatusResult, len(repositories))
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for repository := range jobs {
				results <- e.refreshOne(ctx, repository, activePath)
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, repository := range repositories {
			select {
			case jobs <- repository:
			case <-ctx.Done():
				return
			}
		}
	}()
	group.Wait()
	close(results)
	output := make([]StatusResult, 0, len(repositories))
	for result := range results {
		output = append(output, result)
	}
	return output
}

func (e *Engine) refreshOne(ctx context.Context, repository Repository, activePath string) StatusResult {
	result := StatusResult{Repository: repository}
	e.mu.Lock()
	cached, hasCached := e.cache[repository.Path]
	e.mu.Unlock()
	if hasCached && repository.Path != activePath && e.InactiveAfter > 0 && !repository.LastOpened.IsZero() && time.Since(repository.LastOpened) > e.InactiveAfter {
		cached.Skipped = true
		return cached
	}
	discovery, err := e.Discover(ctx, repository.Path)
	if err == nil {
		result.Snapshot, err = e.Snapshot(ctx, discovery, 0)
	}
	result.Error = err
	e.mu.Lock()
	e.cache[repository.Path] = result
	e.mu.Unlock()
	return result
}
