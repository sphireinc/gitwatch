package provider

import (
	"context"
	"testing"
	"time"
)

func TestCacheInvalidationBypassesUnexpiredData(t *testing.T) {
	cache := NewCache[string](time.Hour)
	calls := 0
	fetch := func(context.Context) (string, error) {
		calls++
		return "fresh", nil
	}
	for i := 0; i < 2; i++ {
		if _, err := cache.Get(context.Background(), "pr", fetch); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("cached calls = %d", calls)
	}
	cache.InvalidateAll()
	if _, err := cache.Get(context.Background(), "pr", fetch); err != nil || calls != 2 {
		t.Fatalf("invalidated cache did not fetch: calls=%d err=%v", calls, err)
	}
}

func TestCacheInvalidationPreventsOldInflightRepopulation(t *testing.T) {
	cache := NewCache[string](time.Hour)
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		_, err := cache.Get(ctx, "pr", func(ctx context.Context) (string, error) {
			close(started)
			select {
			case <-release:
				return "pre-mutation", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		})
		done <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cache.InvalidateAll()
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	called := false
	value, err := cache.Get(ctx, "pr", func(context.Context) (string, error) {
		called = true
		return "post-mutation", nil
	})
	if err != nil || !called || value != "post-mutation" {
		t.Fatalf("old in-flight data repopulated cache: value=%q called=%v err=%v", value, called, err)
	}
}
