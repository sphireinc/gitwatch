package operations

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEngineSerializesRepoAndCancels(t *testing.T) {
	e := New(2)
	started := make(chan struct{})
	release := make(chan struct{})
	if err := e.Submit(context.Background(), "one", "repo", "first", time.Second, func(context.Context) error { close(started); <-release; return nil }); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := e.Submit(context.Background(), "two", "repo", "second", time.Second, func(context.Context) error { return errors.New("should wait") }); err != nil {
		t.Fatal(err)
	}
	e.Cancel("one")
	close(release)
	r := <-e.Results()
	if r.ID != "one" {
		t.Fatal(r)
	}
	r = <-e.Results()
	if r.ID != "two" {
		t.Fatal(r)
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
