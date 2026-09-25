package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fixedToken string

func (t fixedToken) Token() (string, error) { return string(t), nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGitHubClientUsesTokenAndParsesResponses(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer secret-token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		body := ""
		switch {
		case strings.HasSuffix(r.URL.Path, "/pulls"):
			body = `[{"number":7,"title":"Fix","state":"open","html_url":"https://github.com/o/r/pull/7","base":{"ref":"main"},"head":{"ref":"feature"}}]`
		case strings.HasSuffix(r.URL.Path, "/check-runs"):
			body = `{"check_runs":[]}`
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("secret-token"), HTTPClient: &http.Client{Transport: transport}}
	pr, err := client.PullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, "feature")
	if err != nil || pr.Number != 7 {
		t.Fatalf("pull request = %#v, %v", pr, err)
	}
	checks, err := client.Checks(context.Background(), Repository{Owner: "o", Name: "r"}, "abc")
	if err != nil || len(checks.Runs) != 0 {
		t.Fatalf("checks = %#v, %v", checks, err)
	}
}

func TestGitHubClientTreatsMissingBranchPullRequestAsEmpty(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.test", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("head") != "o:feature" || r.URL.Query().Get("state") != "open" {
			t.Fatalf("branch pull request query = %s", r.URL.RawQuery)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("[]")), Header: make(http.Header), Request: r}, nil
	})}}

	pull, err := client.PullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, "feature")
	if err != nil || pull.Number != 0 {
		t.Fatalf("missing branch pull request = %#v, %v", pull, err)
	}
}

func TestGitHubClientListsBoundedOpenPullRequestPage(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("state") != "open" || r.URL.Query().Get("page") != "2" || r.URL.Query().Get("per_page") != "100" {
			t.Fatalf("pagination query = %s", r.URL.RawQuery)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"number":8,"title":"Open"}]`)), Header: make(http.Header), Request: r}, nil
	})}}
	pulls, err := client.ListPullRequests(context.Background(), Repository{Owner: "o", Name: "r"}, 2, 1000)
	if err != nil || len(pulls) != 1 || pulls[0].Number != 8 {
		t.Fatalf("pull page = %#v, err=%v", pulls, err)
	}
}

func TestGitHubClientListsIssuesAndReleasesAndCreatesIssue(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodPost {
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"number":9,"title":"New issue","state":"open"}`)), Header: make(http.Header), Request: r}, nil
		}
		if strings.HasSuffix(r.URL.Path, "/releases") {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":2,"tag_name":"v2.0.0"}]`)), Header: make(http.Header), Request: r}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"number":3,"title":"Bug","state":"open"}]`)), Header: make(http.Header), Request: r}, nil
	})}}
	repository := Repository{Owner: "o", Name: "r"}
	issues, err := client.ListIssues(context.Background(), repository, "bad", 0, MaxIssues+1)
	if err != nil || len(issues) != 1 || issues[0].Number != 3 {
		t.Fatalf("issues = %#v, err=%v", issues, err)
	}
	releases, err := client.ListReleases(context.Background(), repository, 0, MaxReleases+1)
	if err != nil || len(releases) != 1 || releases[0].TagName != "v2.0.0" {
		t.Fatalf("releases = %#v, err=%v", releases, err)
	}
	issue, err := client.CreateIssue(context.Background(), repository, IssueCreateRequest{Title: "New issue", Labels: []string{"bug"}})
	if err != nil || issue.Number != 9 {
		t.Fatalf("created issue = %#v, err=%v", issue, err)
	}
}

func TestGitHubClientLoadsBoundedPullRequestDetail(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"number":9,"title":"Detail","state":"open","base":{"ref":"main"},"head":{"ref":"feature"}}`
		switch {
		case strings.HasSuffix(r.URL.Path, "/commits"):
			body = `[{"sha":"abc","commit":{"message":"subject","author":{"name":"A"}}}]`
		case strings.HasSuffix(r.URL.Path, "/files"):
			body = `[{"filename":"main.go","status":"modified","patch":"@@"}]`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
	})}}
	detail, err := client.PullRequestDetail(context.Background(), Repository{Owner: "o", Name: "r"}, 9)
	if err != nil || detail.Number != 9 || len(detail.Commits) != 1 || len(detail.Files) != 1 {
		t.Fatalf("detail = %#v, err=%v", detail, err)
	}
}

func TestGitHubClientCreatesPullRequestWithoutRetryingMutation(t *testing.T) {
	attempts := 0
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), Retries: 3, HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("request = %s auth=%q", r.Method, r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || !strings.Contains(string(body), `"base":"main"`) || !strings.Contains(string(body), `"head":"feature"`) {
			t.Fatalf("request body = %s, err=%v", body, err)
		}
		return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"number":10,"title":"Ship","state":"open"}`)), Header: make(http.Header), Request: r}, nil
	})}}
	pull, err := client.CreatePullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, PullRequestCreateRequest{Title: "Ship", Body: "body", Head: "feature", Base: "main"})
	if err != nil || pull.Number != 10 || attempts != 1 {
		t.Fatalf("created pull = %#v err=%v attempts=%d", pull, err, attempts)
	}
}

func TestGitHubClientRejectsInvalidPullRequestCreation(t *testing.T) {
	client := GitHubClient{HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid request reached provider")
		return nil, nil
	})}}
	if _, err := client.CreatePullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, PullRequestCreateRequest{Head: "feature", Base: "main"}); err == nil {
		t.Fatal("invalid create request was accepted")
	}
}

func TestGitHubClientReviewAndMergeMutationsAreTypedAndNonRetrying(t *testing.T) {
	attempts := 0
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), Retries: 3, HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("request = %s auth=%q", r.Method, r.Header.Get("Authorization"))
		}
		if strings.HasSuffix(r.URL.Path, "/comments") {
			return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"id":3,"body":"looks good","user":{"login":"reviewer"}}`)), Header: make(http.Header), Request: r}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"merged":true,"sha":"abc","message":"merged"}`)), Header: make(http.Header), Request: r}, nil
	})}}
	comment, err := client.CreateReviewComment(context.Background(), Repository{Owner: "o", Name: "r"}, 4, ReviewCommentRequest{Body: "looks good"})
	if err != nil || comment.ID != 3 {
		t.Fatalf("comment = %#v, err=%v", comment, err)
	}
	merged, err := client.MergePullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, 4, MergeRequest{Method: MergeMethodSquash, ExpectedSHA: "abc"})
	if err != nil || !merged.Merged || merged.SHA != "abc" || attempts != 2 {
		t.Fatalf("merge = %#v, err=%v attempts=%d", merged, err, attempts)
	}
}

func TestGitHubClientDeletesValidatedBranchWithoutRetrying(t *testing.T) {
	attempts := 0
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), Retries: 3, HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if r.Method != http.MethodDelete || r.URL.Path != "/repos/o/r/git/refs/heads/feature/topic" || r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("delete request = %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
	})}}
	if err := client.DeleteBranch(context.Background(), Repository{Owner: "o", Name: "r"}, "feature/topic"); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("delete attempts = %d, want 1", attempts)
	}
	if err := client.DeleteBranch(context.Background(), Repository{Owner: "o", Name: "r"}, "../escape"); err == nil {
		t.Fatal("invalid branch reached provider")
	}
}

func TestGitHubClientDoesNotRetryAmbiguousBranchDelete(t *testing.T) {
	attempts := 0
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), Retries: 3, HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
	})}}
	if err := client.DeleteBranch(context.Background(), Repository{Owner: "o", Name: "r"}, "feature/topic"); err == nil {
		t.Fatal("failed branch deletion unexpectedly succeeded")
	}
	if attempts != 1 {
		t.Fatalf("delete attempts = %d, want 1", attempts)
	}
}

func TestGitHubClientSubmitsReviewDecision(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/reviews") || r.Method != http.MethodPost {
			t.Fatalf("review request path/method = %s %s", r.Method, r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":7,"state":"APPROVED","body":""}`)), Header: make(http.Header), Request: r}, nil
	})}}
	result, err := client.SubmitReview(context.Background(), Repository{Owner: "o", Name: "r"}, 4, ReviewSubmission{Event: ReviewEventApprove, CommitID: "abc"})
	if err != nil || result.ID != 7 || result.State != "APPROVED" {
		t.Fatalf("review result = %#v, err=%v", result, err)
	}
}

func TestGitHubClientRunsCheckActionsWithoutRetrying(t *testing.T) {
	attempts := 0
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), Retries: 3, HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("action request = %s auth=%q", r.Method, r.Header.Get("Authorization"))
		}
		if !strings.HasSuffix(r.URL.Path, "/rerun-failed-jobs") && !strings.HasSuffix(r.URL.Path, "/cancel") {
			t.Fatalf("unexpected action path: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
	})}}
	repository := Repository{Owner: "o", Name: "r"}
	if err := client.RerunFailedJobs(context.Background(), repository, 12); err != nil {
		t.Fatal(err)
	}
	if err := client.CancelRun(context.Background(), repository, 12); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("action attempts = %d, want 2", attempts)
	}
	if err := client.CancelRun(context.Background(), repository, 0); err == nil {
		t.Fatal("invalid check-run id was accepted")
	}
}

func TestGitHubClientReviews(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.github.test", TokenSource: fixedToken("token"), HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"state":"APPROVED"}]`)), Header: make(http.Header)}, nil
	})}}
	reviews, err := client.Reviews(context.Background(), Repository{Owner: "o", Name: "r"}, 3)
	if err != nil || reviews.State() != "approved" {
		t.Fatalf("reviews = %#v, err=%v", reviews, err)
	}
}

func TestGitHubClientDoesNotExposeTokenOnHTTPFailure(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("secret-token")), Header: make(http.Header), Request: r}, nil
	})
	_, err := (GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("secret-token"), HTTPClient: &http.Client{Transport: transport}}).PullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, "main")
	if err == nil || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("unsafe provider error: %v", err)
	}
}

func TestGitHubClientClassifiesRateLimitAndPreservesRetryHint(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		response := &http.Response{StatusCode: http.StatusTooManyRequests, Body: io.NopCloser(strings.NewReader(`{"message":"rate limit"}`)), Header: make(http.Header), Request: r}
		response.Header.Set("Retry-After", "60")
		return response, nil
	})}}
	_, err := client.Checks(context.Background(), Repository{Owner: "o", Name: "r"}, "main")
	var httpErr *HTTPError
	if !errors.Is(err, ErrRateLimited) || !errors.As(err, &httpErr) || httpErr.RetryAfter != "60" {
		t.Fatalf("rate limit error = %v (%T)", err, err)
	}
}

func TestGitHubClientPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := GitHubClient{BaseURL: "https://api.test", HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.Canceled
	})}}
	_, err := client.Checks(ctx, Repository{Owner: "o", Name: "r"}, "main")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestGitHubClientDegradesWhenOffline(t *testing.T) {
	client := GitHubClient{BaseURL: "https://api.test", HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network unreachable")
	})}}
	_, err := client.PullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, "main")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("offline error = %v", err)
	}
}

func TestProviderClassifiesStatesAndRetriesSafeReads(t *testing.T) {
	if Classify(context.Background(), ErrNoToken) != StateNotConfigured || Classify(context.Background(), ErrRateLimited) != StateRateLimited {
		t.Fatal("provider state classification failed")
	}
	attempts := 0
	client := GitHubClient{BaseURL: "https://api.test", Retries: 1, HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header), Request: r}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"check_runs":[]}`)), Header: make(http.Header), Request: r}, nil
	})}}
	if _, err := client.Checks(context.Background(), Repository{Owner: "o", Name: "r"}, "main"); err != nil || attempts != 2 {
		t.Fatalf("retry attempts=%d err=%v", attempts, err)
	}
}

func TestProviderRetryDelayHonorsBoundedRetryAfterForms(t *testing.T) {
	seconds := &http.Response{Header: http.Header{"Retry-After": []string{"30"}}}
	if got := providerRetryDelay(seconds, 0); got != 30*time.Second {
		t.Fatalf("delta retry delay = %s, want 30s", got)
	}
	tooLong := &http.Response{Header: http.Header{"Retry-After": []string{"3600"}}}
	if got := providerRetryDelay(tooLong, 0); got != maxProviderRetryDelay {
		t.Fatalf("bounded retry delay = %s, want %s", got, maxProviderRetryDelay)
	}
	future := time.Now().Add(20 * time.Second)
	date := &http.Response{Header: http.Header{"Retry-After": []string{future.UTC().Format(http.TimeFormat)}}}
	got := providerRetryDelay(date, 0)
	if got < 18*time.Second || got > 20*time.Second {
		t.Fatalf("HTTP-date retry delay = %s, want about 20s", got)
	}
}
