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
	engine.Budget = 100 * time.Millisecond
	engine.Stashes, engine.Remotes = nil, nil
	engine.Discover = func(context.Context, string) (git.Discovery, error) { return git.Discovery{}, nil }
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	repositories := make([]Repository, 128)
	for i := range repositories {
		repositories[i].Path = fmt.Sprintf("repo-%d", i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Refresh(context.Background(), repositories, "active")
	}
}
