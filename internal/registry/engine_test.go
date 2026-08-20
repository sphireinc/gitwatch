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
	entries := []Repository{{Path: "one"}, {Path: "two"}, {Path: "three"}}
	if got := engine.Refresh(context.Background(), entries, "one"); len(got) != 3 || peak.Load() > 2 {
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
