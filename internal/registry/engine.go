package registry

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
	"github.com/sphireinc/git-watch/internal/repo"
)

// StatusResult contains one repository's refresh outcome.
type StatusResult struct {
	Repository Repository
	Snapshot   repo.Snapshot
	Stashes    int
	Remotes    int
	Error      error
	Skipped    bool
	SkipReason string
	Warnings   []string
	// OperationFailed is supplied by operation orchestration when a recent
	// repository-scoped operation failed before the next snapshot. It is kept
	// separate from Error so repository health and operation attention remain
	// distinguishable.
	OperationFailed   bool
	ProviderAttention bool
	Duration          time.Duration
	Refreshed         time.Time
	Gitignore         GitignoreHealth
}

// Engine refreshes repositories concurrently with a bounded worker pool.
type Engine struct {
	Workers          int
	InactiveAfter    time.Duration
	Budget           time.Duration
	InactiveAfterFor func(Repository) time.Duration
	Discover         func(context.Context, string) (git.Discovery, error)
	Snapshot         func(context.Context, git.Discovery, uint64) (repo.Snapshot, error)
	Stashes          func(context.Context, git.Discovery) (int, error)
	Remotes          func(context.Context, git.Discovery) (int, error)
	mu               sync.Mutex
	cache            map[string]StatusResult
}

// NewEngine creates a registry engine with at least one worker.
func NewEngine(workers int) *Engine {
	if workers < 1 {
		workers = 1
	}
	return &Engine{Workers: workers, InactiveAfter: 5 * time.Minute, Budget: 15 * time.Second, Discover: git.Discover, Snapshot: git.Snapshot, Stashes: stashCount, Remotes: remoteCount, cache: make(map[string]StatusResult)}
}

// Refresh reads all repositories and returns results in input order.
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
	inactiveAfter := e.InactiveAfter
	if e.InactiveAfterFor != nil {
		inactiveAfter = e.InactiveAfterFor(repository)
	}
	if hasCached && repository.Path != activePath && inactiveAfter > 0 && !repository.LastOpened.IsZero() && time.Since(repository.LastOpened) > inactiveAfter {
		cached.Skipped = true
		cached.SkipReason = "inactive repository"
		return cached
	}
	if e.Budget > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.Budget)
		defer cancel()
	}
	started := time.Now()
	discovery, err := e.Discover(ctx, repository.Path)
	if err == nil {
		result.Snapshot, err = e.Snapshot(ctx, discovery, 0)
		if err == nil && e.Stashes != nil {
			result.Stashes, err = e.Stashes(ctx, discovery)
			if err != nil {
				result.Warnings = append(result.Warnings, "stash summary: "+err.Error())
				err = nil
			}
		}
		if err == nil && e.Remotes != nil {
			result.Remotes, err = e.Remotes(ctx, discovery)
			if err != nil {
				result.Warnings = append(result.Warnings, "remote summary: "+err.Error())
				err = nil
			}
		}
	}
	result.Error = err
	if err == nil {
		result.Gitignore = inspectGitignore(discovery.Root)
	}
	result.Duration = time.Since(started)
	result.Refreshed = time.Now()
	e.mu.Lock()
	e.cache[repository.Path] = result
	e.mu.Unlock()
	return result
}

func inspectGitignore(root string) GitignoreHealth {
	path := filepath.Join(root, ".gitignore")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return GitignoreHealth{}
	}
	if err != nil {
		return GitignoreHealth{Attention: 1}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return GitignoreHealth{Exists: true, Attention: 1}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return GitignoreHealth{Exists: true, Attention: 1}
	}
	doc, err := document.Parse(data)
	if err != nil {
		return GitignoreHealth{Exists: true, Attention: 1}
	}
	cat, err := catalog.Default()
	if err != nil {
		return GitignoreHealth{Exists: true, Attention: 1}
	}
	health := GitignoreHealth{Exists: true}
	for _, result := range match.Match(doc, cat) {
		switch result.Kind {
		case domain.ManagedExact:
			health.Managed++
		case domain.UnmanagedFull:
			health.Unmanaged++
		case domain.Partial:
			health.Partial++
		case domain.ManagedEdited, domain.InvalidManagedBlock:
			health.Attention++
		}
		if result.UpdateAvailable {
			health.Updates++
		}
	}
	return health
}

func stashCount(ctx context.Context, discovery git.Discovery) (int, error) {
	result, err := git.NewRunner(discovery.Root).Run(ctx, "stash", "list", "--format=%H")
	if err != nil {
		return 0, err
	}
	if len(strings.TrimSpace(string(result.Stdout))) == 0 {
		return 0, nil
	}
	return len(strings.Split(strings.TrimSpace(string(result.Stdout)), "\n")), nil
}

func remoteCount(ctx context.Context, discovery git.Discovery) (int, error) {
	result, err := git.NewRunner(discovery.Root).Run(ctx, "remote")
	if err != nil {
		return 0, err
	}
	if len(strings.TrimSpace(string(result.Stdout))) == 0 {
		return 0, nil
	}
	return len(strings.Split(strings.TrimSpace(string(result.Stdout)), "\n")), nil
}
