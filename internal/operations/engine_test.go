package operations

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEngineSerializesRepoAndCancels(t *testing.T) {
	e := New(2)
	started := make(chan struct{})
	release := make(chan struct{})
	var firstActive atomic.Bool
	if err := e.Submit(context.Background(), "one", "repo", "first", time.Second, func(context.Context) error {
		firstActive.Store(true)
		close(started)
		<-release
		firstActive.Store(false)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	waitErr := errors.New("waited")
	overlapped := make(chan struct{}, 1)
	if err := e.Submit(context.Background(), "two", "repo", "second", time.Second, func(context.Context) error {
		if firstActive.Load() {
			overlapped <- struct{}{}
		}
		return waitErr
	}); err != nil {
		t.Fatal(err)
	}
	e.Cancel("one")
	close(release)
	results := make(map[string]Result, 2)
	for range 2 {
		select {
		case result := <-e.Results():
			results[result.ID] = result
		case <-time.After(time.Second):
			t.Fatal("operation result was not delivered")
		}
	}
	select {
	case <-overlapped:
		t.Fatal("operations for one repository overlapped")
	default:
	}
	if results["one"].State != Cancelled {
		t.Fatalf("first result = %#v", results["one"])
	}
	if results["two"].State != Failed || !errors.Is(results["two"].Err, waitErr) {
		t.Fatalf("second result = %#v", results["two"])
	}
}

func TestEngineBoundsOutstandingOperationsAndReleasesCancelledCapacity(t *testing.T) {
	e := New(1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseWorkers := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseWorkers()
	started := make(chan struct{}, 1)
	work := func(ctx context.Context) error {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for index := 0; index < MaxOutstandingOperations; index++ {
		id := fmt.Sprintf("bounded-%03d", index)
		if err := e.Submit(context.Background(), id, "repo-"+id, "blocked", time.Minute, work); err != nil {
			t.Fatalf("submit %q: %v", id, err)
		}
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("no operation entered the bounded worker pool")
	}
	if err := e.Submit(context.Background(), "overflow", "repo-overflow", "blocked", time.Minute, work); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("overflow submit error = %v, want %v", err, ErrQueueFull)
	}
	var overflowWorkCalls atomic.Int32
	overflow := e.Command(context.Background(), "overflow-command", "repo-overflow", "blocked", time.Minute, func(context.Context) error {
		overflowWorkCalls.Add(1)
		return nil
	})()
	if overflow.Result.State != Failed || !errors.Is(overflow.Result.Err, ErrQueueFull) {
		t.Fatalf("overflow command result = %+v", overflow.Result)
	}
	if overflowWorkCalls.Load() != 0 {
		t.Fatal("work ran despite the outstanding-operation cap")
	}

	if !e.Cancel("bounded-063") {
		t.Fatal("queued operation was not cancellable")
	}
	waitOutstandingOperations(t, e, MaxOutstandingOperations-1)
	if err := e.Submit(context.Background(), "replacement", "repo-replacement", "blocked", time.Minute, work); err != nil {
		t.Fatalf("submit after cancellation: %v", err)
	}
	releaseWorkers()
	waitOutstandingOperations(t, e, 0)

	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.repos) != 0 || len(e.repoRefs) != 0 || len(e.waiters) != 0 {
		t.Fatalf("outstanding-operation cleanup: repos=%d refs=%d waiters=%d", len(e.repos), len(e.repoRefs), len(e.waiters))
	}
	if len(e.latest) > retainedHistoryLimit || len(e.retry) > retainedHistoryLimit {
		t.Fatalf("completed-operation retention: latest=%d retry=%d", len(e.latest), len(e.retry))
	}
}

func waitOutstandingOperations(t *testing.T, engine *Engine, want int) {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		engine.mu.Lock()
		got := len(engine.active)
		engine.mu.Unlock()
		if got == want {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("outstanding operations = %d, want %d", got, want)
		case <-time.After(time.Millisecond):
		}
	}
}

func TestEngineTimeout(t *testing.T) {
	e := New(1)
	if err := e.Submit(context.Background(), "slow", "repo", "slow", time.Millisecond, func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }); err != nil {
		t.Fatal(err)
	}
	r := <-e.Results()
	if r.State != TimedOut {
		t.Fatal(r)
	}
}

func TestCommandsReceiveTheirOwnResultsConcurrently(t *testing.T) {
	e := New(2)
	first := e.Command(context.Background(), "first", "repo-a", "first", time.Second, func(context.Context) error {
		return nil
	})
	second := e.Command(context.Background(), "second", "repo-b", "second", time.Second, func(context.Context) error {
		return nil
	})
	results := make(chan ResultMsg, 2)
	go func() { results <- first() }()
	go func() { results <- second() }()
	seen := map[string]bool{}
	for range 2 {
		select {
		case message := <-results:
			seen[message.Result.ID] = true
		case <-time.After(time.Second):
			t.Fatal("command result was not delivered")
		}
	}
	if !seen["first"] || !seen["second"] {
		t.Fatalf("missing command result: %#v", seen)
	}
}

func TestLifecycleSnapshotClassifiesCancellationAndRetainsHistory(t *testing.T) {
	e := New(1)
	started := make(chan struct{})
	if err := e.Submit(context.Background(), "cancel", "repo", "fetch", time.Second, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	if !e.Cancel("cancel") {
		t.Fatal("active operation was not cancellable")
	}
	result := <-e.Results()
	if result.State != Cancelled || result.Cause == "" || result.Queued.IsZero() || result.Finished.IsZero() {
		t.Fatalf("incomplete cancellation lifecycle: %#v", result)
	}
	if got := e.Snapshot(); len(got) != 1 || got[0].ID != "cancel" || got[0].State != Cancelled {
		t.Fatalf("lifecycle history = %#v", got)
	}
}

func TestRetryRequiresExplicitReplayableOperation(t *testing.T) {
	e := New(1)
	if err := e.Submit(context.Background(), "unsafe", "repo", "merge", time.Second, func(context.Context) error { return errors.New("conflict") }); err != nil {
		t.Fatal(err)
	}
	<-e.Results()
	if _, err := e.Retry(context.Background(), "unsafe"); !errors.Is(err, ErrNotRetryable) {
		t.Fatalf("retry error = %v", err)
	}
}

func TestRetryReplaysExplicitlyMarkedOperation(t *testing.T) {
	e := New(1)
	var attempts atomic.Int32
	if err := e.SubmitWithOptions(context.Background(), "fetch", "repo", "fetch", time.Second, func(context.Context) error {
		if attempts.Add(1) == 1 {
			return errors.New("temporary network failure")
		}
		return nil
	}, Options{Retryable: true}); err != nil {
		t.Fatal(err)
	}
	<-e.Results()
	waiter, err := e.Retry(context.Background(), "fetch")
	if err != nil {
		t.Fatal(err)
	}
	result := <-waiter
	if result.State != Succeeded || attempts.Load() != 2 || !result.Retryable {
		t.Fatalf("retry result = %#v, attempts = %d", result, attempts.Load())
	}
}

func TestEngineBoundsCompletedOperationRetention(t *testing.T) {
	e := New(2)
	oldID := ""
	for index := 0; index < retainedHistoryLimit+8; index++ {
		id := "operation-" + time.Now().Format("150405.000000000") + "-" + string(rune('a'+index))
		if index == 0 {
			oldID = id
		}
		if err := e.SubmitWithOptions(context.Background(), id, "repo", "fetch", time.Second, func(context.Context) error { return nil }, Options{Retryable: true}); err != nil {
			t.Fatal(err)
		}
		result := <-e.Results()
		if result.State != Succeeded {
			t.Fatalf("operation %q = %#v", id, result)
		}
	}
	if got := len(e.Snapshot()); got != retainedHistoryLimit {
		t.Fatalf("retained operation history = %d, want %d", got, retainedHistoryLimit)
	}
	if _, err := e.Retry(context.Background(), oldID); !errors.Is(err, ErrNotRetryable) {
		t.Fatalf("evicted retry error = %v", err)
	}
}

func TestEngineReleasesRepositoryLocksAfterOperationsFinish(t *testing.T) {
	e := New(4)
	for index := 0; index < 64; index++ {
		id := "lock-operation-" + time.Now().Format("150405.000000000") + "-" + string(rune('a'+index))
		repo := "repository-" + string(rune('a'+index))
		if err := e.Submit(context.Background(), id, repo, "refresh", time.Second, func(context.Context) error { return nil }); err != nil {
			t.Fatal(err)
		}
		if result := <-e.Results(); result.State != Succeeded {
			t.Fatalf("operation %q = %#v", id, result)
		}
	}
	if len(e.repos) != 0 || len(e.repoRefs) != 0 {
		t.Fatalf("repository lock retention = repos=%d refs=%d", len(e.repos), len(e.repoRefs))
	}
}
