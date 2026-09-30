package app

import (
	"context"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/provider"
)

func TestCheckRunIdentityCannotAuthorizeWorkflowAction(t *testing.T) {
	m := New()
	m.GitHub.Checks = provider.ChecksSnapshot{Runs: []provider.CheckRun{{ID: 999, Status: "completed", Conclusion: "failure"}}}
	if cmd := m.startGitHubCheckAction("rerun"); cmd != nil || m.GitHubCheckActionConfirm || !strings.Contains(m.Status, "check runs alone") {
		t.Fatalf("check-run ID authorized Actions request: confirmation=%v status=%q", m.GitHubCheckActionConfirm, m.Status)
	}
	m.GitHub.SetWorkflows([]provider.WorkflowRun{{ID: 304, Name: "CI", Status: "completed", Conclusion: "failure", Attempt: 3}})
	m.startGitHubCheckAction("rerun")
	if !m.GitHubCheckActionConfirm || m.GitHubCheckActionRunID != 304 || !strings.Contains(m.Status, "#304, attempt 3") {
		t.Fatalf("workflow target = %d, confirmation=%v status=%q", m.GitHubCheckActionRunID, m.GitHubCheckActionConfirm, m.Status)
	}
}

func TestWorkflowActionLateResultDoesNotAffectAnotherRepository(t *testing.T) {
	m := New()
	m.repositoryGeneration = 2
	m.Status = "current repository"
	updated, cmd := m.Update(GitHubCheckActionFinishedMsg{Generation: 1, Action: "rerun"})
	got := updated.(Model)
	if cmd != nil || got.Status != m.Status {
		t.Fatal("late workflow result refreshed or changed the wrong repository")
	}
}

func TestWorkflowActionSuccessInvalidatesPreActionCache(t *testing.T) {
	m := New()
	ctx := context.Background()
	if _, err := m.GitHubWorkflowsCache.Get(ctx, "head", func(context.Context) ([]provider.WorkflowRun, error) {
		return []provider.WorkflowRun{{ID: 304, Status: "completed"}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	updated, cmd := m.Update(GitHubCheckActionFinishedMsg{Action: "rerun"})
	got := updated.(Model)
	if cmd == nil {
		t.Fatal("successful workflow mutation must schedule provider reload")
	}
	fetched := false
	_, err := got.GitHubWorkflowsCache.Get(ctx, "head", func(context.Context) ([]provider.WorkflowRun, error) {
		fetched = true
		return []provider.WorkflowRun{{ID: 304, Status: "queued"}}, nil
	})
	if err != nil || !fetched {
		t.Fatal("reload could reuse the pre-action workflow cache")
	}
}
