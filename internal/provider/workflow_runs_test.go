package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseWorkflowRunsKeepsActionsIdentityAndAttempt(t *testing.T) {
	runs, err := ParseWorkflowRuns([]byte(`{"workflow_runs":[{"id":304,"check_suite_id":42,"name":"CI","head_sha":"abc","head_branch":"main","status":"completed","conclusion":"failure","run_attempt":3,"html_url":"https://github.com/o/r/actions/runs/304","run_started_at":"2026-09-28T00:00:00Z","updated_at":"2026-09-28T00:02:00Z"}]}`))
	if err != nil || len(runs) != 1 {
		t.Fatalf("runs = %#v, error = %v", runs, err)
	}
	run := runs[0]
	if run.ID != 304 || run.CheckSuiteID != 42 || run.Attempt != 3 || run.HeadSHA != "abc" || run.HeadBranch != "main" || run.Elapsed(time.Now()) != 2*time.Minute {
		t.Fatalf("workflow identity/metadata lost: %#v", run)
	}
}

func TestParseWorkflowRunsRejectsMalformedOrOversizedPages(t *testing.T) {
	for _, input := range []string{`{}`, `{"workflow_runs":null}`, `{"workflow_runs":[{"id":0}]}`, `{"workflow_runs":[{"id":1,"run_attempt":-1}]}`} {
		if _, err := ParseWorkflowRuns([]byte(input)); err == nil {
			t.Errorf("accepted malformed input %s", input)
		}
	}
	items := make([]string, MaxWorkflowRuns+1)
	for i := range items {
		items[i] = fmt.Sprintf(`{"id":%d}`, i+1)
	}
	if _, err := ParseWorkflowRuns([]byte(`{"workflow_runs":[` + strings.Join(items, ",") + `]}`)); err == nil {
		t.Fatal("accepted oversized workflow page")
	}
	if runs, err := ParseWorkflowRuns([]byte(`{"workflow_runs":[]}`)); err != nil || len(runs) != 0 {
		t.Fatalf("empty page = %#v, %v", runs, err)
	}
}

func TestWorkflowElapsedIsObservedAndNonnegative(t *testing.T) {
	now := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name string
		run  WorkflowRun
		want time.Duration
	}{
		{"queued", WorkflowRun{Status: "queued"}, 0},
		{"running", WorkflowRun{Status: "in_progress", StartedAt: now.Add(-time.Minute)}, time.Minute},
		{"future", WorkflowRun{Status: "in_progress", StartedAt: now.Add(time.Minute)}, 0},
		{"completed missing observation", WorkflowRun{Status: "completed", StartedAt: now.Add(-time.Minute)}, 0},
		{"completed negative", WorkflowRun{Status: "completed", StartedAt: now, UpdatedAt: now.Add(-time.Minute)}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.run.Elapsed(now); got != test.want {
				t.Fatalf("elapsed = %v, want %v", got, test.want)
			}
		})
	}
}

func TestGitHubListWorkflowRunsFiltersExactCommitAndBoundsPage(t *testing.T) {
	requests := 0
	client := GitHubClient{BaseURL: "https://api.test", TokenSource: fixedToken("token"), HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/repos/o/r/actions/runs" || r.URL.Query().Get("head_sha") != "abc&branch=other" || r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") != "1" {
			t.Fatalf("workflow request = %s %s", r.Method, r.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"workflow_runs":[{"id":304,"run_attempt":2}]}`)), Request: r}, nil
	})}}
	runs, err := client.ListWorkflowRuns(context.Background(), Repository{Owner: "o", Name: "r"}, "abc&branch=other", 0, 1000)
	if err != nil || len(runs) != 1 || runs[0].ID != 304 {
		t.Fatalf("workflow list = %#v, %v", runs, err)
	}
	if _, err := client.ListWorkflowRuns(context.Background(), Repository{Owner: "o", Name: "r"}, "", 1, 1); err == nil || requests != 1 {
		t.Fatal("empty commit must fail without loading unrelated runs")
	}
}
