package githubview

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/provider"
)

func TestViewRendersPullRequestAndChecksSafely(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Title: "Improve", State: "open", Head: "feature", Base: "main", Mergeable: "clean", URL: "https://github.com/octo/repo/pull/4"}, provider.ChecksSnapshot{Passing: 2, Failing: 1, Pending: 1, Runs: []provider.CheckRun{{Name: "build", Status: "completed", Conclusion: "failure", URL: "https://github.com/check/1", Failure: "compile failed"}}})
	view := m.View()
	for _, want := range []string{"octo/repo", "PR #4", "Checks: 2 passing  1 failing  1 pending", "check build", "compile failed", "https://github.com/check/1"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
	unsafe := New()
	unsafe.SetError(provider.Repository{Owner: "octo\x1b", Name: "repo"}, "main", nil)
	if strings.Contains(unsafe.View(), "\x1b") {
		t.Fatal("provider error view contains escape")
	}
}

func TestViewRendersBoundedPullRequestListAndDetail(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Title: "Improve", State: "open"}, provider.ChecksSnapshot{})
	m.SetPullRequests([]provider.PullRequest{{Number: 4, Title: "Improve", State: "open"}, {Number: 5, Title: "Docs", State: "open"}})
	m.SetDetail(provider.PullRequestDetail{PullRequest: provider.PullRequest{Number: 4, Title: "Improve", State: "open"}, Commits: []provider.PullRequestCommit{{SHA: "abc"}}, Files: []provider.PullRequestFile{{Path: "main.go", Status: "modified", Additions: 2, Deletions: 1}}})
	view := m.View()
	for _, want := range []string{"Open pull requests: 2", "PR #5: Docs", "Commits: 1  Files: 1", "modified main.go +2 -1"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
}

func TestViewRendersSanitizedReviewComments(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Title: "Improve", State: "open"}, provider.ChecksSnapshot{})
	m.SetComments([]provider.ReviewComment{{ID: 1, Author: "reviewer\x1b", Path: "main.go", Line: 4, Body: "please fix\x1b[2J"}})
	view := m.View()
	if strings.Contains(view, "\x1b") || !strings.Contains(view, "Review comments: 1") || !strings.Contains(view, "please fix") {
		t.Fatalf("comments view = %q", view)
	}
}

func TestViewShowsProviderStateAndRetryAfter(t *testing.T) {
	m := New()
	m.SetError(provider.Repository{Owner: "octo", Name: "repo"}, "main", &provider.HTTPError{Status: http.StatusTooManyRequests, RetryAfter: "30"})
	view := m.View()
	for _, want := range []string{"provider state: rate-limited", "retry after 30"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
	if !errors.Is((&provider.HTTPError{Status: http.StatusTooManyRequests}), provider.ErrRateLimited) {
		t.Fatal("rate-limit error no longer unwraps to provider state")
	}
}
