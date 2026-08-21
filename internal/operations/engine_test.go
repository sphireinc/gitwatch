package operations

import (
	"context"
	"errors"
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
