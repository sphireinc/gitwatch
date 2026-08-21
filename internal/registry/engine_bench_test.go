package registry

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/repo"
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
