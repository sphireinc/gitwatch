package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/provider"
	"github.com/sphireinc/git-watch/internal/ui/githubview"
)

type mergePreflightToken string

func (t mergePreflightToken) Token() (string, error) { return string(t), nil }

type mergePreflightHTTP struct {
	state       string
	mergeable   string
	draft       bool
	headSHA     string
	checksBody  string
	checksCode  int
	reviewsBody string
	reviewsCode int
}

func TestGitHubMergePreflightFetchesFreshCriticalStateWithoutCache(t *testing.T) {
	freshSHA := strings.Repeat("a", 40)
	client, requests := newMergePreflightHTTPClient(t, mergePreflightHTTP{
		state:       "open",
		mergeable:   "clean",
		headSHA:     freshSHA,
		checksBody:  `{"total_count":0,"check_runs":[]}`,
		reviewsBody: `[]`,
	})

	m := mergePreflightModel()
	m.GitHub.Pull.HeadSHA = strings.Repeat("b", 40) // deliberately stale workspace state

	for attempt := 0; attempt < 2; attempt++ {
		msg := runMergePreflight(t, m.githubMergePreflightWithClient(&client))
		if msg.Generation != m.repositoryGeneration || msg.PullNumber != 17 {
			t.Fatalf("message scope = generation %d, PR %d", msg.Generation, msg.PullNumber)
		}
		if !msg.CanConfirm || len(msg.Failures) != 0 {
			t.Fatalf("fresh preflight unexpectedly blocked: %#v", msg.Failures)
		}
		if msg.State.Detail.HeadSHA != freshSHA {
			t.Fatalf("head SHA = %q, want fresh %q", msg.State.Detail.HeadSHA, freshSHA)
		}
	}

	for _, path := range []string{
		"/repos/octo/repo/pulls/17",
		"/repos/octo/repo/pulls/17/commits",
		"/repos/octo/repo/pulls/17/files",
		"/repos/octo/repo/commits/" + freshSHA + "/check-runs",
		"/repos/octo/repo/pulls/17/reviews",
	} {
		if requests[path] != 2 {
			t.Errorf("GET %s calls = %d, want 2 direct reads", path, requests[path])
		}
	}
	if requests["/repos/octo/repo/commits/"+strings.Repeat("b", 40)+"/check-runs"] != 0 {
		t.Fatal("checks were not requested using the fresh PR head SHA")
	}
}

func TestGitHubMergePreflightFailsClosedOnCriticalState(t *testing.T) {
	tests := []struct {
		name      string
		response  mergePreflightHTTP
		wantCode  GitHubMergePreflightFailureCode
		wantFetch bool
	}{
		{
			name: "failed checks",
			response: mergePreflightHTTP{state: "open", mergeable: "clean", headSHA: strings.Repeat("a", 40),
				checksBody: `{"total_count":1,"check_runs":[{"id":1,"name":"tests","status":"completed","conclusion":"failure"}]}`, reviewsBody: `[]`},
			wantCode: GitHubMergePreflightChecksFailing, wantFetch: true,
		},
		{
			name: "incomplete checks hide unseen failures",
			response: mergePreflightHTTP{state: "open", mergeable: "clean", headSHA: strings.Repeat("a", 40),
				checksBody: incompleteFailedCheckPage(), reviewsBody: `[]`},
			wantCode: GitHubMergePreflightChecksUnavailable, wantFetch: true,
		},
		{
			name: "pending checks",
			response: mergePreflightHTTP{state: "open", mergeable: "clean", headSHA: strings.Repeat("a", 40),
				checksBody: `{"total_count":1,"check_runs":[{"id":1,"name":"tests","status":"in_progress"}]}`, reviewsBody: `[]`},
			wantCode: GitHubMergePreflightChecksPending, wantFetch: true,
		},
		{
			name: "changes requested",
			response: mergePreflightHTTP{state: "open", mergeable: "clean", headSHA: strings.Repeat("a", 40),
				checksBody: `{"total_count":0,"check_runs":[]}`, reviewsBody: `[{"state":"CHANGES_REQUESTED"}]`},
			wantCode: GitHubMergePreflightChangesRequested, wantFetch: true,
		},
		{
			name: "stale mergeability is not clean",
			response: mergePreflightHTTP{state: "open", mergeable: "behind", headSHA: strings.Repeat("a", 40),
				checksBody: `{"total_count":0,"check_runs":[]}`, reviewsBody: `[]`},
			wantCode: GitHubMergePreflightMergeabilityUnknown, wantFetch: true,
		},
		{
			name: "reviews permission denied",
			response: mergePreflightHTTP{state: "open", mergeable: "clean", headSHA: strings.Repeat("a", 40),
				checksBody: `{"total_count":0,"check_runs":[]}`, reviewsCode: http.StatusForbidden},
			wantCode: GitHubMergePreflightReviewsUnavailable, wantFetch: true,
		},
		{
			name: "draft pull",
			response: mergePreflightHTTP{state: "open", mergeable: "clean", draft: true, headSHA: strings.Repeat("a", 40),
				checksBody: `{"total_count":0,"check_runs":[]}`, reviewsBody: `[]`},
			wantCode: GitHubMergePreflightDraft, wantFetch: true,
		},
		{
			name: "closed pull",
			response: mergePreflightHTTP{state: "closed", mergeable: "clean", headSHA: strings.Repeat("a", 40),
				checksBody: `{"total_count":0,"check_runs":[]}`, reviewsBody: `[]`},
			wantCode: GitHubMergePreflightNotOpen, wantFetch: true,
		},
		{
			name:     "head SHA missing",
			response: mergePreflightHTTP{state: "open", mergeable: "clean", headSHA: "not-a-sha", reviewsBody: `[]`},
			wantCode: GitHubMergePreflightMissingHeadSHA, wantFetch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, requests := newMergePreflightHTTPClient(t, tt.response)
			msg := runMergePreflight(t, mergePreflightModel().githubMergePreflightWithClient(&client))
			if msg.CanConfirm {
				t.Fatal("unsafe preflight enabled merge confirmation")
			}
			if !hasPreflightFailure(msg, tt.wantCode) {
				t.Fatalf("failures %#v do not include %q", msg.Failures, tt.wantCode)
			}
			if tt.wantFetch && requests["/repos/octo/repo/pulls/17"] != 1 {
				t.Fatalf("fresh pull detail calls = %d, want 1", requests["/repos/octo/repo/pulls/17"])
			}
		})
	}
}

func TestGitHubMergePreflightBoundsFailuresAndRejectsInvalidTarget(t *testing.T) {
	m := mergePreflightModel()
	m.GitHub.Pull.Number = 0
	msg := runMergePreflight(t, m.githubMergePreflightWithClient(nil))
	if msg.CanConfirm || msg.Generation != m.repositoryGeneration || msg.PullNumber != 0 {
		t.Fatalf("invalid-target result = %#v", msg)
	}
	if len(msg.Failures) != 1 || msg.Failures[0].Code != GitHubMergePreflightInvalidTarget {
		t.Fatalf("invalid-target failures = %#v", msg.Failures)
	}
}

func TestInvalidateGitHubMutationCaches(t *testing.T) {
	m := mergePreflightModel()
	m.Snapshot.Branch.Name = "main"
	m.GitHubCache = provider.NewPullRequestCache(time.Hour)
	m.GitHubPullsCache = provider.NewCache[[]provider.PullRequest](time.Hour)
	m.GitHubDetailsCache = provider.NewCache[provider.PullRequestDetail](time.Hour)
	m.GitHubCommentsCache = provider.NewCache[[]provider.ReviewComment](time.Hour)
	m.GitHubChecksCache = provider.NewCache[provider.ChecksSnapshot](time.Hour)
	m.GitHubReviewsCache = provider.NewCache[provider.ReviewSnapshot](time.Hour)
	m.GitHubIssuesCache = provider.NewCache[[]provider.Issue](time.Hour)
	m.GitHubReleasesCache = provider.NewCache[[]provider.Release](time.Hour)
	m.GitHubWorkflowsCache = provider.NewCache[[]provider.WorkflowRun](time.Hour)

	calls := make(map[string]int)
	ctx := context.Background()
	_, _ = m.GitHubPullsCache.Get(ctx, "open", func(context.Context) ([]provider.PullRequest, error) { calls["pulls"]++; return nil, nil })
	_, _ = m.GitHubDetailsCache.Get(ctx, "detail", func(context.Context) (provider.PullRequestDetail, error) {
		calls["details"]++
		return provider.PullRequestDetail{}, nil
	})
	_, _ = m.GitHubCommentsCache.Get(ctx, "comments", func(context.Context) ([]provider.ReviewComment, error) { calls["comments"]++; return nil, nil })
	_, _ = m.GitHubChecksCache.Get(ctx, "checks", func(context.Context) (provider.ChecksSnapshot, error) {
		calls["checks"]++
		return provider.ChecksSnapshot{}, nil
	})
	_, _ = m.GitHubReviewsCache.Get(ctx, "reviews", func(context.Context) (provider.ReviewSnapshot, error) {
		calls["reviews"]++
		return provider.ReviewSnapshot{}, nil
	})
	_, _ = m.GitHubIssuesCache.Get(ctx, "issues", func(context.Context) ([]provider.Issue, error) { calls["issues"]++; return nil, nil })
	_, _ = m.GitHubReleasesCache.Get(ctx, "releases", func(context.Context) ([]provider.Release, error) { calls["releases"]++; return nil, nil })
	_, _ = m.GitHubWorkflowsCache.Get(ctx, "workflows", func(context.Context) ([]provider.WorkflowRun, error) { calls["workflows"]++; return nil, nil })
	pullClient := mergePreflightPullClient{calls: &calls}
	_, _ = m.GitHubCache.Get(ctx, pullClient, m.GitHub.Repository, "main")

	m.invalidateGitHubMutationCaches()
	_, _ = m.GitHubPullsCache.Get(ctx, "open", func(context.Context) ([]provider.PullRequest, error) { calls["pulls"]++; return nil, nil })
	_, _ = m.GitHubDetailsCache.Get(ctx, "detail", func(context.Context) (provider.PullRequestDetail, error) {
		calls["details"]++
		return provider.PullRequestDetail{}, nil
	})
	_, _ = m.GitHubCommentsCache.Get(ctx, "comments", func(context.Context) ([]provider.ReviewComment, error) { calls["comments"]++; return nil, nil })
	_, _ = m.GitHubChecksCache.Get(ctx, "checks", func(context.Context) (provider.ChecksSnapshot, error) {
		calls["checks"]++
		return provider.ChecksSnapshot{}, nil
	})
	_, _ = m.GitHubReviewsCache.Get(ctx, "reviews", func(context.Context) (provider.ReviewSnapshot, error) {
		calls["reviews"]++
		return provider.ReviewSnapshot{}, nil
	})
	_, _ = m.GitHubIssuesCache.Get(ctx, "issues", func(context.Context) ([]provider.Issue, error) { calls["issues"]++; return nil, nil })
	_, _ = m.GitHubReleasesCache.Get(ctx, "releases", func(context.Context) ([]provider.Release, error) { calls["releases"]++; return nil, nil })
	_, _ = m.GitHubWorkflowsCache.Get(ctx, "workflows", func(context.Context) ([]provider.WorkflowRun, error) { calls["workflows"]++; return nil, nil })
	_, _ = m.GitHubCache.Get(ctx, pullClient, m.GitHub.Repository, "main")

	for _, name := range []string{"pulls", "details", "comments", "checks", "reviews", "issues", "releases", "workflows", "branch"} {
		if calls[name] != 2 {
			t.Errorf("%s fetches = %d, want 2 across mutation invalidation", name, calls[name])
		}
	}
}

type mergePreflightPullClient struct{ calls *map[string]int }

func (c mergePreflightPullClient) PullRequest(_ context.Context, _ provider.Repository, branch string) (provider.PullRequest, error) {
	(*c.calls)["branch"]++
	return provider.PullRequest{Number: 17, Head: branch}, nil
}

func mergePreflightModel() Model {
	return Model{
		repositoryGeneration: 9,
		GitHub:               githubModelForMergePreflight(),
	}
}

func githubModelForMergePreflight() githubview.Model {
	return githubview.Model{
		Repository: provider.Repository{Host: "github.com", Owner: "octo", Name: "repo"},
		Branch:     "main",
		Pull:       provider.PullRequest{Number: 17, HeadSHA: strings.Repeat("b", 40)},
	}
}

func mergePreflightClient(httpClient *http.Client) provider.GitHubClient {
	return provider.GitHubClient{BaseURL: "https://api.github.test", TokenSource: mergePreflightToken("test-token"), HTTPClient: httpClient, Retries: 0}
}

func newMergePreflightHTTPClient(t *testing.T, response mergePreflightHTTP) (provider.GitHubClient, map[string]int) {
	t.Helper()
	requests := make(map[string]int)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("preflight made unexpected %s request to %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/octo/repo/pulls/17":
			if _, err := fmt.Fprintf(w, `{"number":17,"title":"Improve","state":%q,"draft":%t,"head":{"ref":"topic","sha":%q},"base":{"ref":"main"},"mergeable_state":%q}`, response.state, response.draft, response.headSHA, response.mergeable); err != nil {
				t.Errorf("write pull request response: %v", err)
			}
		case "/repos/octo/repo/pulls/17/commits", "/repos/octo/repo/pulls/17/files":
			_, _ = w.Write([]byte(`[]`))
		case "/repos/octo/repo/commits/" + strings.Repeat("a", 40) + "/check-runs":
			if r.URL.Query().Get("filter") != "latest" || r.URL.Query().Get("per_page") != "100" {
				t.Errorf("check-run query = %s", r.URL.RawQuery)
			}
			writePreflightResponse(w, response.checksCode, response.checksBody)
		case "/repos/octo/repo/pulls/17/reviews":
			if r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") == "" {
				t.Errorf("reviews pagination query = %s", r.URL.RawQuery)
			}
			writePreflightResponse(w, response.reviewsCode, response.reviewsBody)
		default:
			http.NotFound(w, r)
		}
	})
	transport := mergePreflightRoundTripper(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		return recorder.Result(), nil
	})
	return mergePreflightClient(&http.Client{Transport: transport}), requests
}

type mergePreflightRoundTripper func(*http.Request) (*http.Response, error)

func (f mergePreflightRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func writePreflightResponse(w http.ResponseWriter, status int, body string) {
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if body != "" {
		_, _ = w.Write([]byte(body))
	}
}

func incompleteFailedCheckPage() string {
	run := `{"id":1,"name":"tests","status":"completed","conclusion":"failure"}`
	return `{"total_count":101,"check_runs":[` + strings.TrimSuffix(strings.Repeat(run+",", provider.MaxCheckRuns), ",") + `]}`
}

func runMergePreflight(t *testing.T, command tea.Cmd) GitHubMergePreflightMsg {
	t.Helper()
	if command == nil {
		t.Fatal("preflight command is nil")
	}
	result := command()
	msg, ok := result.(GitHubMergePreflightMsg)
	if !ok {
		t.Fatalf("preflight returned %T", result)
	}
	return msg
}

func hasPreflightFailure(msg GitHubMergePreflightMsg, code GitHubMergePreflightFailureCode) bool {
	for _, failure := range msg.Failures {
		if failure.Code == code {
			return true
		}
	}
	return false
}
