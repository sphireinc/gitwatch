package registry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/repo"
)

func BenchmarkRefreshInjectedSlowSources(b *testing.B) {
	engine := NewEngine(4)
	engine.Budget = time.Second
	engine.Stashes, engine.Remotes = nil, nil
	engine.Discover = delayedDiscovery(1 * time.Millisecond)
	engine.Snapshot = delayedSnapshot(1 * time.Millisecond)
	repositories := make([]Repository, 128)
	for i := range repositories {
		repositories[i].Path = fmt.Sprintf("repo-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Refresh(context.Background(), repositories, "active")
	}
}

func BenchmarkRefreshInjectedNetworkLatency(b *testing.B) {
	engine := NewEngine(4)
	engine.Budget = time.Second
	engine.Stashes, engine.Remotes = nil, nil
	engine.Discover = delayedDiscovery(2 * time.Millisecond)
	engine.Snapshot = delayedSnapshot(2 * time.Millisecond)
	repositories := make([]Repository, 64)
	for i := range repositories {
		repositories[i].Path = fmt.Sprintf("remote-repo-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Refresh(context.Background(), repositories, "active")
	}
}

func BenchmarkRefreshRepositories(b *testing.B) {
	for _, size := range []int{10, 50, 100} {
		for _, scenario := range []string{"cold", "warm", "cache-hit", "mixed-failure"} {
			b.Run(fmt.Sprintf("%d-repositories/%s", size, scenario), func(b *testing.B) {
				benchmarkRefreshRepositories(b, size, scenario)
			})
		}
	}
}

func benchmarkRefreshRepositories(b *testing.B, size int, scenario string) {
	repositories := refreshBenchmarkRepositories(size, scenario == "cache-hit")
	failurePaths := make(map[string]struct{}, (size+9)/10)
	if scenario == "mixed-failure" {
		for index := 0; index < size; index += 10 {
			failurePaths[repositories[index].Path] = struct{}{}
		}
	}
	newEngine := func() *Engine {
		engine := NewEngine(8)
		engine.Stashes, engine.Remotes, engine.Worktrees, engine.CommitConfig = nil, nil, nil, nil
		engine.Discover = func(_ context.Context, path string) (git.Discovery, error) {
			if _, fail := failurePaths[path]; fail {
				return git.Discovery{}, fmt.Errorf("synthetic discovery failure")
			}
			return git.Discovery{}, nil
		}
		engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) {
			return repo.Snapshot{}, nil
		}
		if scenario == "cache-hit" {
			engine.InactiveAfter = time.Minute
		}
		return engine
	}

	var engine *Engine
	activePath := "repo-0"
	switch scenario {
	case "warm":
		engine = newEngine()
		_ = engine.Refresh(context.Background(), repositories, activePath)
	case "cache-hit":
		engine = newEngine()
		_ = engine.Refresh(context.Background(), repositories, activePath)
		activePath = ""
	case "cold", "mixed-failure":
	default:
		b.Fatalf("unknown refresh benchmark scenario %q", scenario)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		iterationEngine := engine
		if scenario == "cold" || scenario == "mixed-failure" {
			iterationEngine = newEngine()
		}
		results := iterationEngine.Refresh(context.Background(), repositories, activePath)
		if len(results) != size {
			b.Fatalf("refresh returned %d repositories, want %d", len(results), size)
		}
		failures := 0
		for _, result := range results {
			if result.Error != nil {
				failures++
			}
			if scenario == "cache-hit" && !result.Skipped {
				b.Fatalf("inactive repository %q was not served by the production cache path", result.Repository.Path)
			}
		}
		wantFailures := 0
		if scenario == "mixed-failure" {
			wantFailures = (size + 9) / 10
		}
		if failures != wantFailures {
			b.Fatalf("refresh failures = %d, want %d for %s", failures, wantFailures, scenario)
		}
	}
}

func refreshBenchmarkRepositories(size int, stale bool) []Repository {
	repositories := make([]Repository, size)
	for index := range repositories {
		repositories[index] = Repository{Path: fmt.Sprintf("repo-%d", index)}
		if stale {
			repositories[index].LastOpened = time.Now().Add(-time.Hour)
		}
	}
	return repositories
}

func delayedDiscovery(delay time.Duration) func(context.Context, string) (git.Discovery, error) {
	return func(ctx context.Context, _ string) (git.Discovery, error) {
		if err := waitForBenchmarkDelay(ctx, delay); err != nil {
			return git.Discovery{}, err
		}
		return git.Discovery{}, nil
	}
}

func delayedSnapshot(delay time.Duration) func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) {
	return func(ctx context.Context, _ git.Discovery, _ uint64) (repo.Snapshot, error) {
		if err := waitForBenchmarkDelay(ctx, delay); err != nil {
			return repo.Snapshot{}, err
		}
		return repo.Snapshot{}, nil
	}
}

func waitForBenchmarkDelay(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
