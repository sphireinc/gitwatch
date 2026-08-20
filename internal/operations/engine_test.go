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
