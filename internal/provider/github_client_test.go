package provider

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
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
