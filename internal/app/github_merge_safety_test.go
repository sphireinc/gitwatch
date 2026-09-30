package app

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/provider"
	"github.com/sphireinc/git-watch/internal/workspace"
)

func TestCachedGitHubLoadCannotEnableMergeConfirmation(t *testing.T) {
	m := New()
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	pull := provider.PullRequest{Number: 4, State: "open", HeadSHA: strings.Repeat("a", 40), Mergeable: "clean"}
	m.GitHub.SetData(provider.Repository{Owner: "o", Name: "r"}, "main", pull, provider.ChecksSnapshot{})
	m.GitHubMergeRefresh = true
	updated, _ := m.Update(GitHubReadyMsg{Repository: m.GitHub.Repository, Branch: "main", Pull: pull, ProviderStale: true, Checks: provider.ChecksSnapshot{Passing: 1}, Review: provider.ReviewSnapshot{Approved: 1}})
	m = updated.(Model)
	if m.GitHubMergeConfirm {
		t.Fatal("ordinary/cached workspace load must not authorize a merge")
	}
	updated, _ = m.Update(GitHubMergePreflightMsg{PullNumber: 4, CanConfirm: false, Failures: []GitHubMergePreflightFailure{{Code: GitHubMergePreflightChecksUnavailable, Err: errors.New("checks unavailable")}}})
	m = updated.(Model)
	if m.GitHubMergeConfirm || m.GitHubMergeRefresh || !strings.Contains(m.Status, "preflight blocked") {
		t.Fatalf("failed preflight did not fail closed: %q", m.Status)
	}
}

func TestLateGitHubMutationResultsCannotChangeAnotherRepository(t *testing.T) {
	for _, msg := range []tea.Msg{
		GitHubMergeFinishedMsg{Generation: 1, Result: provider.MergeResult{Merged: true}},
		GitHubReviewFinishedMsg{Generation: 1},
		GitHubReviewCommentFinishedMsg{Generation: 1},
		GitHubBranchDeleteFinishedMsg{Generation: 1, Branch: "old"},
	} {
		m := New()
		m.repositoryGeneration = 2
		m.State, m.Status = StateReady, "current repository"
		updated, cmd := m.Update(msg)
		got := updated.(Model)
		if cmd != nil || got.Status != m.Status || got.GitHubBranchDeleteConfirm {
			t.Fatalf("late %T changed a different repository", msg)
		}
	}
}
