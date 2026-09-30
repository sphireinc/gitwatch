package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"
)

const MaxWorkflowRuns = 100

// WorkflowRun is an Actions workflow execution, not a check-run or job ID.
// Only this ID may be sent to the workflow rerun/cancel endpoints.
type WorkflowRun struct {
	ID           int64
	CheckSuiteID int64
	Name         string
	Status       string
	Conclusion   string
	URL          string
	HeadSHA      string
	HeadBranch   string
	Attempt      int
	StartedAt    time.Time
	UpdatedAt    time.Time
}

// Elapsed returns the observed runtime. For completed runs GitHub's updated_at
// is the available end observation, not a sum of individual job durations.
func (r WorkflowRun) Elapsed(now time.Time) time.Duration {
	if r.StartedAt.IsZero() {
		return 0
	}
	end := now
	if r.Status == "completed" {
		if r.UpdatedAt.IsZero() {
			return 0
		}
		end = r.UpdatedAt
	}
	return max(0, end.Sub(r.StartedAt))
}

type WorkflowRunListClient interface {
	ListWorkflowRuns(context.Context, Repository, string, int, int) ([]WorkflowRun, error)
}

func ParseWorkflowRuns(data []byte) ([]WorkflowRun, error) {
	var response struct {
		Runs []struct {
			ID           int64      `json:"id"`
			CheckSuiteID int64      `json:"check_suite_id"`
			Name         string     `json:"name"`
			Status       string     `json:"status"`
			Conclusion   string     `json:"conclusion"`
			URL          string     `json:"html_url"`
			HeadSHA      string     `json:"head_sha"`
			HeadBranch   string     `json:"head_branch"`
			Attempt      int        `json:"run_attempt"`
			StartedAt    *time.Time `json:"run_started_at"`
			UpdatedAt    *time.Time `json:"updated_at"`
		} `json:"workflow_runs"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	if response.Runs == nil {
		return nil, errors.New("invalid workflow runs response")
	}
	if len(response.Runs) > MaxWorkflowRuns {
		return nil, errors.New("workflow run page exceeds bound")
	}
	runs := make([]WorkflowRun, 0, len(response.Runs))
	for _, raw := range response.Runs {
		if raw.ID < 1 || raw.Attempt < 0 {
			return nil, errors.New("invalid workflow run identity or attempt")
		}
		run := WorkflowRun{ID: raw.ID, CheckSuiteID: raw.CheckSuiteID, Name: raw.Name, Status: raw.Status, Conclusion: raw.Conclusion, URL: raw.URL, HeadSHA: raw.HeadSHA, HeadBranch: raw.HeadBranch, Attempt: raw.Attempt}
		if raw.StartedAt != nil {
			run.StartedAt = raw.StartedAt.UTC()
		}
		if raw.UpdatedAt != nil {
			run.UpdatedAt = raw.UpdatedAt.UTC()
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// ListWorkflowRuns loads one bounded page for an exact commit. It never falls
// back to listing unrelated repository runs when no commit is available.
func (c GitHubClient) ListWorkflowRuns(ctx context.Context, repository Repository, headSHA string, page, perPage int) ([]WorkflowRun, error) {
	if headSHA == "" {
		return nil, errors.New("workflow runs require a head commit")
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > MaxWorkflowRuns {
		perPage = MaxWorkflowRuns
	}
	query := url.Values{"head_sha": {headSHA}, "page": {fmt.Sprint(page)}, "per_page": {fmt.Sprint(perPage)}}
	path := "/repos/" + url.PathEscape(repository.Owner) + "/" + url.PathEscape(repository.Name) + "/actions/runs?" + query.Encode()
	var response json.RawMessage
	if err := c.getJSON(ctx, path, &response); err != nil {
		return nil, err
	}
	return ParseWorkflowRuns(response)
}
