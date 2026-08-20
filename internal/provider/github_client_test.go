package provider

import (
	"context"
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

func TestGitHubClientDoesNotExposeTokenOnHTTPFailure(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("secret-token")), Header: make(http.Header), Request: r}, nil
	})
	_, err := (GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("secret-token"), HTTPClient: &http.Client{Transport: transport}}).PullRequest(context.Background(), Repository{Owner: "o", Name: "r"}, "main")
	if err == nil || strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("unsafe provider error: %v", err)
	}
}
