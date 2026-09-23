package remoteintel

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/provider"
)

func TestRunOnceBoundsWorkersAndSkipsActiveRepositories(t *testing.T) {
	scheduler := New(Config{Enabled: true, Interval: time.Hour, Workers: 2})
	repositories := make([]Repository, 0, 20)
	for index := 0; index < 20; index++ {
		repositories = append(repositories, Repository{Path: string(rune('a' + index))})
	}
	repositories[0].ActiveOperation = true
	var running, maximum atomic.Int32
	results := scheduler.RunOnce(context.Background(), repositories, time.Unix(100, 0), func(ctx context.Context, path string) error {
		current := running.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		select {
		case <-time.After(time.Millisecond):
		case <-ctx.Done():
		}
		running.Add(-1)
		return nil
	})
	if maximum.Load() > 2 {
		t.Fatalf("maximum concurrent fetches = %d", maximum.Load())
	}
	fetched, skipped := 0, 0
	for _, result := range results {
		switch result.Status {
		case "fetched":
			fetched++
		case "skipped-active":
			skipped++
		}
	}
	if fetched != 19 || skipped != 1 {
		t.Fatalf("results fetched=%d skipped=%d: %#v", fetched, skipped, results)
	}
}

func TestRunOnceBacksOffFailuresAndHonorsCancellation(t *testing.T) {
	now := time.Unix(200, 0)
	scheduler := New(Config{Enabled: true, Interval: time.Hour, BackoffBase: 10 * time.Second, BackoffMax: time.Minute, Workers: 1})
	fail := errors.New("offline")
	results := scheduler.RunOnce(context.Background(), []Repository{{Path: "/repo"}}, now, func(context.Context, string) error { return fail })
	if len(results) != 1 || results[0].Status != "failed" || !errors.Is(results[0].Err, fail) || !results[0].Next.Equal(now.Add(10*time.Second)) {
		t.Fatalf("first failure = %#v", results)
	}
	if results = scheduler.RunOnce(context.Background(), []Repository{{Path: "/repo"}}, now.Add(time.Second), func(context.Context, string) error { t.Fatal("backoff should skip fetch"); return nil }); len(results) != 0 {
		t.Fatalf("backoff results = %#v", results)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	results = scheduler.RunOnce(ctx, []Repository{{Path: "/other"}}, now, func(context.Context, string) error { t.Fatal("cancelled fetch should not run"); return nil })
	if len(results) != 0 {
		t.Fatalf("cancelled results = %#v", results)
	}
}

func TestRunOnceSkipsActiveOperationReturnedByFetch(t *testing.T) {
	scheduler := New(Config{Enabled: true, Interval: time.Hour, Workers: 1})
	results := scheduler.RunOnce(context.Background(), []Repository{{Path: "/repo"}}, time.Unix(1, 0), func(context.Context, string) error { return ErrActiveOperation })
	if len(results) != 1 || results[0].Status != "skipped-active" || results[0].Err != nil {
		t.Fatalf("active operation result = %#v", results)
	}
}

func TestClassifyErrorUsesStableRemoteCategories(t *testing.T) {
	for _, test := range []struct {
		message string
		want    string
	}{
		{"fatal: unable to access: network is offline", "offline"},
		{"remote: HTTP 429 too many requests", "rate-limited"},
		{"authentication required: 403", "authentication"},
		{"remote rejected by policy", "remote-error"},
	} {
		if got := ClassifyError(errors.New(test.message)); got != test.want {
			t.Fatalf("ClassifyError(%q) = %q, want %q", test.message, got, test.want)
		}
	}
}

func TestClassifyErrorUsesTypedProviderQuotaAndPermissionStates(t *testing.T) {
	quota := &provider.HTTPError{Status: http.StatusForbidden, RateLimitRemaining: "0"}
	if got := ClassifyError(quota); got != "rate-limited" {
		t.Fatalf("typed quota classification = %q, want rate-limited", got)
	}
	permission := &provider.HTTPError{Status: http.StatusForbidden, RateLimitRemaining: "12"}
	if got := ClassifyError(permission); got != "authentication" {
		t.Fatalf("typed permission classification = %q, want authentication", got)
	}
}

func TestRunOnceUsesMostConservativeGroupInterval(t *testing.T) {
	now := time.Unix(300, 0)
	scheduler := New(Config{Enabled: true, Interval: time.Hour, Workers: 1, GroupIntervals: map[string]time.Duration{"fast": 5 * time.Minute, "slow": 20 * time.Minute}})
	results := scheduler.RunOnce(context.Background(), []Repository{{Path: "/repo", Groups: []string{"slow", "fast"}}}, now, func(context.Context, string) error { return nil })
	if len(results) != 1 || !results[0].Next.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("group interval result = %#v", results)
	}
}
