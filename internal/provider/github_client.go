package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var ErrProviderUnavailable = errors.New("GitHub provider unavailable")
var ErrRateLimited = errors.New("GitHub API rate limited")

type HTTPError struct {
	Status     int
	RetryAfter string
}

func (e *HTTPError) Error() string {
	if e.RetryAfter == "" {
		return fmt.Sprintf("GitHub HTTP %d", e.Status)
	}
	return fmt.Sprintf("GitHub HTTP %d; retry after %s", e.Status, e.RetryAfter)
}

func (e *HTTPError) Unwrap() error {
	if e.Status == http.StatusForbidden || e.Status == http.StatusTooManyRequests {
		return ErrRateLimited
	}
	return ErrProviderUnavailable
}

type GitHubClient struct {
	BaseURL     string
	TokenSource TokenSource
	HTTPClient  *http.Client
}

func (c GitHubClient) PullRequest(ctx context.Context, repository Repository, branch string) (PullRequest, error) {
	var values []json.RawMessage
	query := url.Values{"head": []string{repository.Owner + ":" + branch}, "state": []string{"open"}, "per_page": []string{"1"}}
	if err := c.getJSON(ctx, "/repos/"+url.PathEscape(repository.Owner)+"/"+url.PathEscape(repository.Name)+"/pulls?"+query.Encode(), &values); err != nil {
		return PullRequest{}, err
	}
	if len(values) == 0 {
		return PullRequest{}, ErrProviderUnavailable
	}
	return ParsePullRequest(values[0])
}

func (c GitHubClient) Checks(ctx context.Context, repository Repository, ref string) (ChecksSnapshot, error) {
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/commits/" + url.PathEscape(ref) + "/check-runs"
	if err := c.getJSON(ctx, path, &response); err != nil {
		return ChecksSnapshot{}, err
	}
	return ParseChecks(response)
}

func (c GitHubClient) Reviews(ctx context.Context, repository Repository, number int) (ReviewSnapshot, error) {
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls/" + url.PathEscape(fmt.Sprint(number)) + "/reviews"
	if err := c.getJSON(ctx, path, &response); err != nil {
		return ReviewSnapshot{}, err
	}
	return ParseReviews(response)
}

func (c GitHubClient) getJSON(ctx context.Context, path string, target any) error {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	token := ""
	if c.TokenSource != nil {
		value, err := c.TokenSource.Token()
		if err != nil {
			return ErrNoToken
		}
		token = value
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return ErrProviderUnavailable
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "gitwatch")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrProviderUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return &HTTPError{Status: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(target); err != nil {
		return fmt.Errorf("%w: invalid response", ErrProviderUnavailable)
	}
	return nil
}
