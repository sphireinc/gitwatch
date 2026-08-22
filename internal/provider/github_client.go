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
	"time"
)

// ErrProviderUnavailable indicates that the optional provider cannot be used.
var ErrProviderUnavailable = errors.New("GitHub provider unavailable")

// ErrRateLimited indicates that the provider rejected a request for quota.
var ErrRateLimited = errors.New("GitHub API rate limited")

// State describes the bounded availability state of optional provider data.
type State string

const (
	StateDisabled       State = "disabled"
	StateNotConfigured  State = "not configured"
	StateAuthenticating State = "authenticating"
	StateAvailable      State = "available"
	StateStaleCache     State = "stale-cache"
	StateRateLimited    State = "rate-limited"
	StateUnauthorized   State = "unauthorized"
	StateUnavailable    State = "unavailable"
	StateMalformed      State = "malformed"
	StateCanceled       State = "canceled"
)

// HTTPError preserves an unsuccessful provider response and status code.
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

// GitHubClient fetches optional GitHub data using a token source.
type GitHubClient struct {
	BaseURL     string
	TokenSource TokenSource
	HTTPClient  *http.Client
	Timeout     time.Duration
	Retries     int
}

// PullRequest fetches the open pull request for branch.
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

// Checks fetches check runs for ref.
func (c GitHubClient) Checks(ctx context.Context, repository Repository, ref string) (ChecksSnapshot, error) {
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/commits/" + url.PathEscape(ref) + "/check-runs"
	if err := c.getJSON(ctx, path, &response); err != nil {
		return ChecksSnapshot{}, err
	}
	return ParseChecks(response)
}

// Reviews fetches review state for a pull request number.
func (c GitHubClient) Reviews(ctx context.Context, repository Repository, number int) (ReviewSnapshot, error) {
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls/" + url.PathEscape(fmt.Sprint(number)) + "/reviews"
	if err := c.getJSON(ctx, path, &response); err != nil {
		return ReviewSnapshot{}, err
	}
	return ParseReviews(response)
}

func (c GitHubClient) getJSON(ctx context.Context, path string, target any) (returnErr error) {
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
		client = &http.Client{}
	}
	if c.Timeout > 0 {
		copy := *client
		copy.Timeout = c.Timeout
		client = &copy
	}
	response, err := c.doWithRetry(ctx, client, request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrProviderUnavailable
	}
	defer func() {
		if err := response.Body.Close(); err != nil && returnErr == nil {
			returnErr = fmt.Errorf("%w: close response body: %v", ErrProviderUnavailable, err)
		}
	}()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		httpErr := &HTTPError{Status: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}
		if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, 4096)); err != nil {
			return fmt.Errorf("%w: discard response body: %v", httpErr, err)
		}
		return httpErr
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(target); err != nil {
		return fmt.Errorf("%w: invalid response", ErrProviderUnavailable)
	}
	return nil
}

func (c GitHubClient) doWithRetry(ctx context.Context, client *http.Client, request *http.Request) (*http.Response, error) {
	retries := c.Retries
	if retries < 0 {
		retries = 0
	}
	if retries > 3 {
		retries = 3
	}
	for attempt := 0; ; attempt++ {
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		if attempt >= retries || (response.StatusCode < 500 && response.StatusCode != http.StatusTooManyRequests) {
			return response, nil
		}
		_ = response.Body.Close()
		delay := time.Duration(1<<attempt) * 50 * time.Millisecond
		if retryAfter := response.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil && seconds < time.Second {
				delay = seconds
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		request = request.Clone(ctx)
	}
}
