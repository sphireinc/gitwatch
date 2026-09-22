package provider

import (
	"bytes"
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
	Status             int
	RetryAfter         string
	RateLimitRemaining string
	RateLimitReset     string
}

func (e *HTTPError) Error() string {
	if e.RetryAfter == "" {
		return fmt.Sprintf("GitHub HTTP %d", e.Status)
	}
	return fmt.Sprintf("GitHub HTTP %d; retry after %s", e.Status, e.RetryAfter)
}

func (e *HTTPError) Unwrap() error {
	if e.IsRateLimited() {
		return ErrRateLimited
	}
	return ErrProviderUnavailable
}

// IsRateLimited distinguishes GitHub quota responses from permission-denied
// 403 responses. Header values are retained as bounded metadata only; response
// bodies are deliberately never exposed in provider errors.
func (e *HTTPError) IsRateLimited() bool {
	if e == nil {
		return false
	}
	return e.Status == http.StatusTooManyRequests || e.RetryAfter != "" || strings.TrimSpace(e.RateLimitRemaining) == "0"
}

func newHTTPError(response *http.Response) *HTTPError {
	return &HTTPError{
		Status:             response.StatusCode,
		RetryAfter:         response.Header.Get("Retry-After"),
		RateLimitRemaining: response.Header.Get("X-RateLimit-Remaining"),
		RateLimitReset:     response.Header.Get("X-RateLimit-Reset"),
	}
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

// ListPullRequests returns one bounded page of open pull requests.
func (c GitHubClient) ListPullRequests(ctx context.Context, repository Repository, page, perPage int) ([]PullRequest, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	if perPage > MaxPullRequests {
		perPage = MaxPullRequests
	}
	query := url.Values{"state": []string{"open"}, "per_page": []string{fmt.Sprint(perPage)}, "page": []string{fmt.Sprint(page)}}
	var values json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls?" + query.Encode()
	if err := c.getJSON(ctx, path, &values); err != nil {
		return nil, err
	}
	return ParsePullRequests(values)
}

// ListIssues returns one bounded page of repository issues. Pull requests are
// excluded because GitHub exposes them through the issues API as well.
func (c GitHubClient) ListIssues(ctx context.Context, repository Repository, state string, page, perPage int) ([]Issue, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > MaxIssues {
		perPage = MaxIssues
	}
	if state != "open" && state != "closed" && state != "all" {
		state = "open"
	}
	query := url.Values{"state": []string{state}, "per_page": []string{fmt.Sprint(perPage)}, "page": []string{fmt.Sprint(page)}}
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/issues?" + query.Encode()
	if err := c.getJSON(ctx, path, &response); err != nil {
		return nil, err
	}
	return ParseIssues(response)
}

func (c GitHubClient) CreateIssue(ctx context.Context, repository Repository, request IssueCreateRequest) (Issue, error) {
	if err := request.Validate(); err != nil {
		return Issue{}, err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return Issue{}, err
	}
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/issues"
	if err := c.postJSON(ctx, path, payload, &response); err != nil {
		return Issue{}, err
	}
	issues, err := ParseIssues([]byte("[" + string(response) + "]"))
	if err != nil || len(issues) != 1 {
		if err != nil {
			return Issue{}, err
		}
		return Issue{}, errors.New("invalid issue response")
	}
	return issues[0], nil
}

func (c GitHubClient) ListReleases(ctx context.Context, repository Repository, page, perPage int) ([]Release, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > MaxReleases {
		perPage = MaxReleases
	}
	query := url.Values{"per_page": []string{fmt.Sprint(perPage)}, "page": []string{fmt.Sprint(page)}}
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/releases?" + query.Encode()
	if err := c.getJSON(ctx, path, &response); err != nil {
		return nil, err
	}
	return ParseReleases(response)
}

// PullRequestDetail loads bounded PR metadata, commits, and changed files.
func (c GitHubClient) PullRequestDetail(ctx context.Context, repository Repository, number int) (PullRequestDetail, error) {
	if number < 1 {
		return PullRequestDetail{}, errors.New("invalid pull request number")
	}
	base := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls/" + url.PathEscape(fmt.Sprint(number))
	var pullData json.RawMessage
	if err := c.getJSON(ctx, base, &pullData); err != nil {
		return PullRequestDetail{}, err
	}
	pull, err := ParsePullRequest(pullData)
	if err != nil {
		return PullRequestDetail{}, err
	}
	var commitData json.RawMessage
	if err := c.getJSON(ctx, base+"/commits?per_page=100", &commitData); err != nil {
		return PullRequestDetail{}, err
	}
	commits, err := ParsePullRequestCommits(commitData)
	if err != nil {
		return PullRequestDetail{}, err
	}
	var fileData json.RawMessage
	if err := c.getJSON(ctx, base+"/files?per_page=100", &fileData); err != nil {
		return PullRequestDetail{}, err
	}
	files, err := ParsePullRequestFiles(fileData)
	if err != nil {
		return PullRequestDetail{}, err
	}
	return PullRequestDetail{PullRequest: pull, Commits: commits, Files: files}, nil
}

// CreatePullRequest creates only the provider object. It never pushes a local
// branch; callers must use the existing guarded Git push workflow first.
func (c GitHubClient) CreatePullRequest(ctx context.Context, repository Repository, request PullRequestCreateRequest) (PullRequest, error) {
	if strings.TrimSpace(request.Title) == "" || strings.TrimSpace(request.Head) == "" || strings.TrimSpace(request.Base) == "" {
		return PullRequest{}, errors.New("pull request title, head, and base are required")
	}
	if len(request.Title) > MaxPatchBytes || len(request.Body) > MaxPatchBytes || len(request.Head) > 256 || len(request.Base) > 256 {
		return PullRequest{}, errors.New("pull request request exceeds bounds")
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return PullRequest{}, err
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	token := ""
	if c.TokenSource != nil {
		token, err = c.TokenSource.Token()
		if err != nil {
			return PullRequest{}, ErrNoToken
		}
	}
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(payload))
	if err != nil {
		return PullRequest{}, ErrProviderUnavailable
	}
	httpRequest.Header.Set("Accept", "application/vnd.github+json")
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("User-Agent", "gitwatch")
	if token != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+token)
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
	response, err := client.Do(httpRequest)
	if err != nil {
		if ctx.Err() != nil {
			return PullRequest{}, ctx.Err()
		}
		return PullRequest{}, ErrProviderUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return PullRequest{}, newHTTPError(response)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return PullRequest{}, fmt.Errorf("%w: read response", ErrProviderUnavailable)
	}
	return ParsePullRequest(data)
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

// RerunFailedJobs requests a rerun of failed jobs for a check run. The
// mutation is deliberately non-retrying because repeating it can duplicate a
// provider-side action.
func (c GitHubClient) RerunFailedJobs(ctx context.Context, repository Repository, runID int64) error {
	if runID < 1 {
		return errors.New("invalid check run id")
	}
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/actions/runs/" + url.PathEscape(fmt.Sprint(runID)) + "/rerun-failed-jobs"
	return c.postJSON(ctx, path, nil, nil)
}

// CancelRun requests cancellation of a running workflow run. The mutation is
// deliberately non-retrying because repeating it can duplicate a provider-side
// action.
func (c GitHubClient) CancelRun(ctx context.Context, repository Repository, runID int64) error {
	if runID < 1 {
		return errors.New("invalid check run id")
	}
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/actions/runs/" + url.PathEscape(fmt.Sprint(runID)) + "/cancel"
	return c.postJSON(ctx, path, nil, nil)
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

func (c GitHubClient) ListReviewComments(ctx context.Context, repository Repository, number int) ([]ReviewComment, error) {
	if number < 1 {
		return nil, errors.New("invalid pull request number")
	}
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls/" + url.PathEscape(fmt.Sprint(number)) + "/comments?per_page=100"
	if err := c.getJSON(ctx, path, &response); err != nil {
		return nil, err
	}
	return ParseReviewComments(response)
}

func (c GitHubClient) CreateReviewComment(ctx context.Context, repository Repository, number int, request ReviewCommentRequest) (ReviewComment, error) {
	if number < 1 {
		return ReviewComment{}, errors.New("invalid pull request number")
	}
	if err := request.Validate(); err != nil {
		return ReviewComment{}, err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return ReviewComment{}, err
	}
	var response json.RawMessage
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls/" + url.PathEscape(fmt.Sprint(number)) + "/comments"
	if err := c.postJSON(ctx, path, payload, &response); err != nil {
		return ReviewComment{}, err
	}
	comments, err := ParseReviewComments([]byte("[" + string(response) + "]"))
	if err != nil || len(comments) != 1 {
		if err != nil {
			return ReviewComment{}, err
		}
		return ReviewComment{}, errors.New("invalid review comment response")
	}
	return comments[0], nil
}

func (c GitHubClient) SubmitReview(ctx context.Context, repository Repository, number int, submission ReviewSubmission) (ReviewSubmissionResult, error) {
	if number < 1 {
		return ReviewSubmissionResult{}, errors.New("invalid pull request number")
	}
	if err := submission.Validate(); err != nil {
		return ReviewSubmissionResult{}, err
	}
	payload, err := json.Marshal(submission)
	if err != nil {
		return ReviewSubmissionResult{}, err
	}
	var response struct {
		ID      int64  `json:"id"`
		State   string `json:"state"`
		Body    string `json:"body"`
		Message string `json:"message"`
	}
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls/" + url.PathEscape(fmt.Sprint(number)) + "/reviews"
	if err := c.postJSON(ctx, path, payload, &response); err != nil {
		return ReviewSubmissionResult{}, err
	}
	if response.ID < 1 {
		return ReviewSubmissionResult{}, errors.New("invalid review submission response")
	}
	return ReviewSubmissionResult{ID: response.ID, State: response.State, Body: response.Body, Message: response.Message}, nil
}

func (c GitHubClient) MergePullRequest(ctx context.Context, repository Repository, number int, request MergeRequest) (MergeResult, error) {
	if number < 1 {
		return MergeResult{}, errors.New("invalid pull request number")
	}
	if err := request.Validate(); err != nil {
		return MergeResult{}, err
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return MergeResult{}, err
	}
	var response struct {
		Merged  bool   `json:"merged"`
		SHA     string `json:"sha"`
		Message string `json:"message"`
	}
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/pulls/" + url.PathEscape(fmt.Sprint(number)) + "/merge"
	if err := c.postJSON(ctx, path, payload, &response); err != nil {
		return MergeResult{}, err
	}
	return MergeResult{Merged: response.Merged, SHA: response.SHA, Message: response.Message}, nil
}

// DeleteBranch deletes a branch ref through GitHub. The branch name is
// validated before it is interpolated into the API path.
func (c GitHubClient) DeleteBranch(ctx context.Context, repository Repository, branch string) error {
	if err := ValidateCheckoutRef(branch); err != nil {
		return err
	}
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/git/refs/heads/" + url.PathEscape(branch)
	return c.delete(ctx, path)
}

func (c GitHubClient) postJSON(ctx context.Context, path string, payload []byte, target any) error {
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
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(payload))
	if err != nil {
		return ErrProviderUnavailable
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Content-Type", "application/json")
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
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrProviderUnavailable
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return newHTTPError(response)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("%w: read response", ErrProviderUnavailable)
	}
	if err := json.Unmarshal(data, target); err != nil {
		if target == nil && len(data) == 0 {
			return nil
		}
		return fmt.Errorf("%w: invalid response", ErrProviderUnavailable)
	}
	return nil
}

func (c GitHubClient) delete(ctx context.Context, path string) error {
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
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, base+path, nil)
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
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return newHTTPError(response)
	}
	return nil
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
		httpErr := newHTTPError(response)
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
