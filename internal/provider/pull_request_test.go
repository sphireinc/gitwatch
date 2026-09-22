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

func TestParsePullRequestsBoundsPages(t *testing.T) {
	pulls, err := ParsePullRequests([]byte(`[{"number":1,"title":"One"},{"number":2,"title":"Two"}]`))
	if err != nil || len(pulls) != 2 || pulls[1].Number != 2 {
		t.Fatalf("pulls = %#v, err=%v", pulls, err)
	}
	tooMany := make([]byte, 0)
	tooMany = append(tooMany, '[')
	for i := 0; i < MaxPullRequests+1; i++ {
		if i > 0 {
			tooMany = append(tooMany, ',')
		}
		tooMany = append(tooMany, `{"number":1,"title":"One"}`...)
	}
	tooMany = append(tooMany, ']')
	if _, err := ParsePullRequests(tooMany); err == nil {
		t.Fatal("oversized pull request page was accepted")
	}
}

func TestParsePullRequestDetailPartsAreBounded(t *testing.T) {
	commits, err := ParsePullRequestCommits([]byte(`[{"sha":"abc","commit":{"message":"subject","author":{"name":"A"}}}]`))
	if err != nil || len(commits) != 1 || commits[0].Author != "A" {
		t.Fatalf("commits = %#v, err=%v", commits, err)
	}
	files, err := ParsePullRequestFiles([]byte(`[{"filename":"main.go","status":"modified","additions":2,"deletions":1,"changes":3,"patch":"@@ -1 +1 @@"}]`))
	if err != nil || len(files) != 1 || files[0].Path != "main.go" {
		t.Fatalf("files = %#v, err=%v", files, err)
	}
	if _, err := ParsePullRequestCommits([]byte(`[{"sha":""}]`)); err == nil {
		t.Fatal("invalid commit was accepted")
	}
}

func TestValidateCheckoutRefRejectsUntrustedProviderRefs(t *testing.T) {
	for _, ref := range []string{"feature/topic", "release-1"} {
		if err := ValidateCheckoutRef(ref); err != nil {
			t.Fatalf("valid ref %q rejected: %v", ref, err)
		}
	}
	for _, ref := range []string{"", "-bad", "../escape", "bad\x1bref", "bad ref"} {
		if err := ValidateCheckoutRef(ref); err == nil {
			t.Fatalf("unsafe ref %q accepted", ref)
		}
	}
}

func TestParseReviewCommentsBoundsAndFields(t *testing.T) {
	comments, err := ParseReviewComments([]byte(`[{"id":12,"body":"please update","user":{"login":"reviewer"},"path":"main.go","line":4,"side":"RIGHT","diff_hunk":"@@"}]`))
	if err != nil || len(comments) != 1 || comments[0].Author != "reviewer" || comments[0].Line != 4 {
		t.Fatalf("comments = %#v, err=%v", comments, err)
	}
	if err := (ReviewCommentRequest{Body: "comment", Side: "MIDDLE"}).Validate(); err == nil {
		t.Fatal("invalid review side was accepted")
	}
	if err := (MergeRequest{Method: MergeMethod("fast-forward")}).Validate(); err == nil {
		t.Fatal("invalid merge method was accepted")
	}
}

func TestReviewSubmissionValidationRequiresReasonForChanges(t *testing.T) {
	if err := (ReviewSubmission{Event: ReviewEventApprove}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (ReviewSubmission{Event: ReviewEventRequestChanges}).Validate(); err == nil {
		t.Fatal("request changes without body was accepted")
	}
	if err := (ReviewSubmission{Event: ReviewEvent("DISMISS")}).Validate(); err == nil {
		t.Fatal("unknown review event was accepted")
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
