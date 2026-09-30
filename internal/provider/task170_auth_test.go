package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type task170Token string

func (t task170Token) Token() (string, error) { return string(t), nil }

func TestTask170Expired401IsUnauthorizedWithoutLeakingBodyOrToken(t *testing.T) {
	const token = "task170-secret-token"
	client := GitHubClient{
		BaseURL:     "https://api.test",
		TokenSource: task170Token(token),
		HTTPClient: &http.Client{Transport: task170RoundTripFunc(func(request *http.Request) (*http.Response, error) {
			if got := request.Header.Get("Authorization"); got != "Bearer "+token {
				t.Errorf("authorization header = %q", got)
			}
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader("expired credential response must not escape")),
				Header:     make(http.Header),
				Request:    request,
			}, nil
		})},
	}
	_, err := client.ListReleases(context.Background(), Repository{Owner: "octo", Name: "repo"}, 1, 1)
	if err == nil || Classify(context.Background(), err) != StateUnauthorized {
		t.Fatalf("expired 401 classification = state %q, error %v", Classify(context.Background(), err), err)
	}
	if strings.Contains(err.Error(), token) || strings.Contains(err.Error(), "expired credential response") {
		t.Fatalf("expired 401 leaked sensitive data: %v", err)
	}
}

type task170RoundTripFunc func(*http.Request) (*http.Response, error)

func (f task170RoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
