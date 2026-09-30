package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

type pullFetchFunc func(context.Context, Repository, string) (PullRequest, error)

func (f pullFetchFunc) PullRequest(ctx context.Context, repository Repository, branch string) (PullRequest, error) {
	return f(ctx, repository, branch)
}

func TestPullRequestStaleFallbackHasExplicitProvenance(t *testing.T) {
	cache := NewPullRequestCache(time.Hour)
	repository := Repository{Owner: "o", Name: "r"}
	ctx := context.Background()
	if _, err := cache.Get(ctx, pullFetchFunc(func(context.Context, Repository, string) (PullRequest, error) {
		return PullRequest{Number: 4}, nil
	}), repository, "main"); err != nil {
		t.Fatal(err)
	}
	cache.mu.Lock()
	item := cache.items["/o/r@main"]
	item.At = time.Now().Add(-2 * time.Hour)
	cache.items["/o/r@main"] = item
	cache.mu.Unlock()
	value, stale, err := cache.GetWithStale(ctx, pullFetchFunc(func(context.Context, Repository, string) (PullRequest, error) {
		return PullRequest{}, errors.New("unavailable")
	}), repository, "main")
	if err == nil || !stale || value.Number != 4 {
		t.Fatalf("stale fallback = %#v, stale=%v, err=%v", value, stale, err)
	}
	cache.Invalidate(repository, "main")
	value, stale, err = cache.GetWithStale(ctx, pullFetchFunc(func(context.Context, Repository, string) (PullRequest, error) {
		return PullRequest{}, errors.New("unavailable")
	}), repository, "main")
	if err == nil || stale || value.Number != 0 {
		t.Fatal("invalidated PR must not be returned as stale pre-mutation data")
	}
}

func TestPullRequestInvalidationPreventsInflightRepopulation(t *testing.T) {
	cache := NewPullRequestCache(time.Hour)
	repository := Repository{Owner: "o", Name: "r"}
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		_, err := cache.Get(ctx, pullFetchFunc(func(ctx context.Context, _ Repository, _ string) (PullRequest, error) {
			close(started)
			select {
			case <-release:
				return PullRequest{Number: 4, State: "open"}, nil
			case <-ctx.Done():
				return PullRequest{}, ctx.Err()
			}
		}), repository, "main")
		done <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cache.Invalidate(repository, "main")
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	fetched := false
	value, err := cache.Get(ctx, pullFetchFunc(func(context.Context, Repository, string) (PullRequest, error) {
		fetched = true
		return PullRequest{Number: 4, State: "closed"}, nil
	}), repository, "main")
	if err != nil || !fetched || value.State != "closed" {
		t.Fatal("pre-mutation PR load repopulated invalidated cache")
	}
}
