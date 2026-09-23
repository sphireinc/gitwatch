package provider

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadCISummariesBoundsWorkersAndRepositoryCount(t *testing.T) {
	requests := make([]CISummaryRequest, MaxDashboardCIRepositories+5)
	for index := range requests {
		requests[index] = CISummaryRequest{Key: string(rune('a' + index)), Ref: "main"}
	}
	var active, peak atomic.Int32
	results := LoadCISummaries(context.Background(), requests, 12, func(_ context.Context, request CISummaryRequest) CISummary {
		current := active.Add(1)
		defer active.Add(-1)
		for observed := peak.Load(); current > observed; observed = peak.Load() {
			if peak.CompareAndSwap(observed, current) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		return CISummary{Key: request.Key, State: "passing"}
	})
	if len(results) != MaxDashboardCIRepositories {
		t.Fatalf("summary count = %d, want %d", len(results), MaxDashboardCIRepositories)
	}
	if got := peak.Load(); got < 2 || got > DashboardCIWorkers {
		t.Fatalf("peak workers = %d, want 2 through %d", got, DashboardCIWorkers)
	}
	for index, result := range results {
		if result.Key != requests[index].Key || result.State != "passing" {
			t.Fatalf("summary %d = %#v", index, result)
		}
	}
}

func TestLoadCISummariesStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	results := LoadCISummaries(ctx, []CISummaryRequest{{Key: "repo", Ref: "main"}}, 1, func(context.Context, CISummaryRequest) CISummary {
		called = true
		return CISummary{}
	})
	if called || len(results) != 0 {
		t.Fatalf("canceled load called fetch=%v and returned %d summaries", called, len(results))
	}
}
