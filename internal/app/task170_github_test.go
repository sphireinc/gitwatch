package app

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/provider"
	"github.com/sphireinc/git-watch/internal/workspace"
)

func TestTask170PalettePRSelectionSurvivesLoadAndRejectsOlderGeneration(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	initCommittedTestRepository(t, ctx, root, "task170 selection")
	runner := git.NewRunner(root)
	gitMustRunAppTest(t, ctx, runner, "remote", "add", "origin", "https://github.com/octo/repo.git")
	m := NewRepositoryWithConfig(git.Discovery{Root: root}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	defer func() { _ = m.Close() }()
	m.Snapshot.Branch.Name, m.Snapshot.Branch.OID = "main", "local-head-sha"
	m.GitHub.SetData(provider.Repository{Host: "github.com", Owner: "octo", Name: "repo"}, "main", provider.PullRequest{}, provider.ChecksSnapshot{})
	m.GitHub.SetPullRequests([]provider.PullRequest{{Number: 8, Title: "selected", Head: "feature", HeadSHA: "pr-head-sha", Base: "main"}})
	transport := appRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		respond := func(body string) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}, nil
		}
		switch {
		case r.URL.Path == "/repos/octo/repo/pulls" && r.URL.Query().Get("head") != "":
			return respond(`[]`)
		case r.URL.Path == "/repos/octo/repo/pulls":
			return respond(`[{"number":8,"title":"selected","state":"open","head":{"ref":"feature","sha":"pr-head-sha"},"base":{"ref":"main"}}]`)
		case r.URL.Path == "/repos/octo/repo/pulls/8":
			return respond(`{"number":8,"title":"selected detail","state":"open","head":{"ref":"feature","sha":"pr-head-sha"},"base":{"ref":"main"}}`)
		case strings.HasSuffix(r.URL.Path, "/commits"), strings.HasSuffix(r.URL.Path, "/files"), strings.HasSuffix(r.URL.Path, "/comments"), strings.HasSuffix(r.URL.Path, "/reviews"), r.URL.Path == "/repos/octo/repo/issues", r.URL.Path == "/repos/octo/repo/releases":
			return respond(`[]`)
		case strings.HasSuffix(r.URL.Path, "/check-runs"):
			return respond(`{"total_count":0,"check_runs":[]}`)
		case r.URL.Path == "/repos/octo/repo/actions/runs":
			return respond(`{"workflow_runs":[{"id":1701,"check_suite_id":9,"name":"CI","status":"completed","conclusion":"success","head_sha":"pr-head-sha","run_attempt":2}]}`)
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header), Request: r}, nil
		}
	})
	client := provider.GitHubClient{BaseURL: "https://api.test", HTTPClient: &http.Client{Transport: transport}}
	m.GitHubDetailsCache = provider.NewCache[provider.PullRequestDetail](0)
	m.GitHubWorkflowsCache = provider.NewCache[[]provider.WorkflowRun](0)
	if cmd := m.executePaletteAction("palette_pr_0"); cmd == nil || m.currentView() != workspace.GitHub {
		t.Fatalf("palette PR selection did not enter the GitHub workspace: cmd=%v view=%v", cmd != nil, m.currentView())
	}
	msg, ok := m.loadGitHubWithClient(&client)().(GitHubReadyMsg)
	if !ok || msg.Err != nil {
		t.Fatalf("GitHub load = %#v", msg)
	}
	if msg.Pull.Number != 8 || msg.Detail == nil || msg.Detail.Number != 8 || msg.BranchPull.Number != 0 {
		t.Fatalf("selected PR load lost explicit selection or active-branch state: %#v", msg)
	}
	if len(msg.Workflows) != 1 || msg.Workflows[0].ID != 1701 {
		t.Fatalf("workflow runs not loaded for selected PR: %#v", msg.Workflows)
	}
	updated, _ := m.Update(msg)
	m = updated.(Model)
	if m.GitHub.Pull.Number != 8 || m.GitHub.Detail == nil || m.GitHub.Detail.Number != 8 || len(m.GitHub.Workflows) != 1 {
		t.Fatalf("selected PR/detail/workflows not applied: pull=%#v detail=%#v workflows=%#v", m.GitHub.Pull, m.GitHub.Detail, m.GitHub.Workflows)
	}
	late := GitHubReadyMsg{Generation: m.repositoryGeneration, Request: msg.Request - 1, Pull: provider.PullRequest{}, Branch: "main"}
	updated, _ = m.Update(late)
	m = updated.(Model)
	if m.GitHub.Pull.Number != 8 || m.GitHub.Detail == nil || m.GitHub.Detail.Number != 8 {
		t.Fatalf("late load replaced the current selection: pull=%#v detail=%#v", m.GitHub.Pull, m.GitHub.Detail)
	}
	if cmd := m.navigate(workspace.Status, "Status"); cmd != nil || m.githubPRSelection != nil {
		t.Fatalf("leaving GitHub did not clear explicit PR selection: cmd=%v selection=%#v", cmd != nil, m.githubPRSelection)
	}
}

func TestTask170CreateOffersGuardedRemoteFlowWithoutPushing(t *testing.T) {
	m := NewRepositoryWithConfig(git.Discovery{Root: t.TempDir()}, config.Config{GitHub: config.GitHubConfig{Enabled: true}})
	defer func() { _ = m.Close() }()
	m.Workspace.Navigate(workspace.GitHub, "GitHub")
	m.GitHub.SetData(provider.Repository{Host: "github.com", Owner: "octo", Name: "repo"}, "feature", provider.PullRequest{}, provider.ChecksSnapshot{})
	m.Snapshot.Branch.Name = "feature"
	m.Snapshot.Branch.Upstream = ""
	if cmd := m.startGitHubCreate(); cmd != nil || !m.GitHubPushOffer || m.GitHubCreateMode {
		t.Fatalf("missing-upstream create did not offer push flow: cmd=%v offer=%v form=%v", cmd != nil, m.GitHubPushOffer, m.GitHubCreateMode)
	}
	updated, cmd := m.Update(key("y"))
	m = updated.(Model)
	if cmd == nil || m.currentView() != workspace.Remotes || m.GitHubCreateMode || m.RemotePushConfirm {
		t.Fatalf("accepting offer did more than navigate to guarded Remotes flow: cmd=%v view=%v form=%v pushConfirm=%v", cmd != nil, m.currentView(), m.GitHubCreateMode, m.RemotePushConfirm)
	}
}
