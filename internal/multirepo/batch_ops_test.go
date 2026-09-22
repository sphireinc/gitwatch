package multirepo

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

func TestRunValidatesStrategiesAndIsolatesFailures(t *testing.T) {
	requests := []Request{
		{Repository: Repository{ID: "one", Root: "/one"}, Remote: "origin", Action: ActionFetch},
		{Repository: Repository{ID: "two", Root: "/two"}, Remote: "origin", Branch: "main", Strategy: "ff-only", Action: ActionPull},
		{Repository: Repository{ID: "bad", Root: "/bad"}, Remote: "origin", Branch: "main", Action: ActionPull},
	}
	var calls atomic.Int32
	results := Run(context.Background(), requests, 2, func(_ context.Context, request Request) error {
		calls.Add(1)
		if request.Repository.ID == "two" {
			return errors.New("remote rejected")
		}
		return nil
	})
	if results[0].Status != "succeeded" || results[1].Status != "failed" || results[2].Status != "skipped" || calls.Load() != 2 {
		t.Fatalf("results = %#v calls=%d", results, calls.Load())
	}
}

func TestRunCancellationDoesNotStartQueuedRequests(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls atomic.Int32
	requests := make([]Request, 8)
	for i := range requests {
		requests[i] = Request{Repository: Repository{ID: domain.RepositoryID("repo" + string(rune('a'+i))), Root: "/repo"}, Remote: "origin", Action: ActionFetch}
	}
	results := Run(ctx, requests, 1, func(_ context.Context, _ Request) error {
		calls.Add(1)
		cancel()
		return context.Canceled
	})
	if calls.Load() != 1 || results[0].Status != "cancelled" {
		t.Fatalf("calls=%d results=%#v", calls.Load(), results)
	}
}

func TestRunFiftyRequestsKeepsWorkerBound(t *testing.T) {
	requests := make([]Request, 50)
	for index := range requests {
		requests[index] = Request{
			Repository: Repository{ID: domain.RepositoryID("repo-" + string(rune('a'+index))), Root: "/repo"},
			Remote:     "origin",
			Action:     ActionFetch,
		}
	}
	const workers = 4
	var active, peak atomic.Int32
	results := Run(context.Background(), requests, workers, func(context.Context, Request) error {
		current := active.Add(1)
		for {
			previous := peak.Load()
			if current <= previous || peak.CompareAndSwap(previous, current) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		active.Add(-1)
		return nil
	})
	if len(results) != len(requests) || peak.Load() > workers {
		t.Fatalf("batch results=%d peak=%d, want results=%d and peak<=%d", len(results), peak.Load(), len(requests), workers)
	}
	for index, result := range results {
		if result.Status != "succeeded" || result.Request.Repository.ID != requests[index].Repository.ID {
			t.Fatalf("result[%d] = %#v, want successful input order", index, result)
		}
	}
}
