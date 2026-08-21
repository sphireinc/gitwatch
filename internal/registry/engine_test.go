package registry

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/repo"
)

func TestEngineUsesBoundedWorkersAndCachesInactiveRepositories(t *testing.T) {
	var active atomic.Int32
	var peak atomic.Int32
	var calls atomic.Int32
	engine := NewEngine(2)
	engine.InactiveAfter = time.Hour
	engine.Discover = func(context.Context, string) (git.Discovery, error) {
		current := active.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		defer active.Add(-1)
		calls.Add(1)
		return git.Discovery{Root: "repo"}, nil
	}
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = func(context.Context, git.Discovery) (int, error) { return 3, nil }
	engine.Remotes = func(context.Context, git.Discovery) (int, error) { return 2, nil }
	entries := []Repository{{Path: "one"}, {Path: "two"}, {Path: "three"}}
	if got := engine.Refresh(context.Background(), entries, "one"); len(got) != 3 || peak.Load() > 2 || got[0].Stashes != 3 || got[0].Remotes != 2 {
		t.Fatalf("unexpected refresh: len=%d peak=%d", len(got), peak.Load())
	}
	if calls.Load() != 3 {
		t.Fatalf("unexpected discovery count: %d", calls.Load())
	}
	entries[1].LastOpened = time.Now().Add(-2 * time.Hour)
	entries[2].LastOpened = time.Now().Add(-2 * time.Hour)
	engine.Refresh(context.Background(), entries, "one")
	if calls.Load() != 4 {
		t.Fatalf("inactive cache was not used: %d", calls.Load())
	}
}

func TestEngineUsesRepositoryRefreshPolicy(t *testing.T) {
	engine := NewEngine(1)
	engine.InactiveAfter = time.Hour
	engine.InactiveAfterFor = func(repository Repository) time.Duration {
		if len(repository.Groups) > 0 && repository.Groups[0] == "fast" {
			return time.Minute
		}
		return time.Hour
	}
	engine.Discover = func(context.Context, string) (git.Discovery, error) { return git.Discovery{}, nil }
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = nil
	engine.Remotes = nil
	entries := []Repository{{Path: "fast", Groups: []string{"fast"}, LastOpened: time.Now().Add(-2 * time.Hour)}, {Path: "slow", LastOpened: time.Now().Add(-30 * time.Minute)}}
	engine.Refresh(context.Background(), entries, "active")
	results := engine.Refresh(context.Background(), entries, "active")
	if !results[0].Skipped || results[1].Skipped {
		t.Fatalf("refresh policy results = %#v", results)
	}
}
