package provider

import (
	"context"
	"testing"
	"time"
)

func TestParsePullRequestAndCache(t *testing.T) {
	value, err := ParsePullRequest([]byte(`{"number":12,"title":"Fix","state":"open","draft":true,"html_url":"https://github.com/o/r/pull/12","base":{"ref":"main"},"head":{"ref":"feature"},"mergeable_state":"clean"}`))
	if err != nil || value.Number != 12 || !value.Draft || value.Base != "main" {
		t.Fatalf("unexpected pull request: %#v, %v", value, err)
	}
	client := &fakePRClient{value: value}
	cache := NewPullRequestCache(time.Hour)
	repository := Repository{Host: "github.com", Owner: "o", Name: "r"}
	if _, err = cache.Get(context.Background(), client, repository, "feature"); err != nil {
		t.Fatal(err)
	}
	if _, err = cache.Get(context.Background(), client, repository, "feature"); err != nil || client.calls != 1 {
		t.Fatalf("cache miss: calls=%d err=%v", client.calls, err)
	}
}

func TestParseReviewsPrioritizesRequestedChanges(t *testing.T) {
	reviews, err := ParseReviews([]byte(`[{"state":"APPROVED"},{"state":"COMMENTED"},{"state":"CHANGES_REQUESTED"}]`))
	if err != nil || reviews.Approved != 1 || reviews.Commented != 1 || reviews.Changes != 1 || reviews.State() != "changes requested" {
		t.Fatalf("reviews = %#v, err=%v", reviews, err)
	}
}

type fakePRClient struct {
	value PullRequest
	calls int
}

func (f *fakePRClient) PullRequest(context.Context, Repository, string) (PullRequest, error) {
	f.calls++
	return f.value, nil
}
