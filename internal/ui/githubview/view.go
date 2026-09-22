package githubview

import (
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/provider"
)

type Model struct {
	Repository  provider.Repository
	Branch      string
	Pull        provider.PullRequest
	Pulls       []provider.PullRequest
	Issues      []provider.Issue
	Releases    []provider.Release
	Detail      *provider.PullRequestDetail
	Comments    []provider.ReviewComment
	Checks      provider.ChecksSnapshot
	SelectedRun int
	Ready       bool
	Error       string
}

func New() Model { return Model{} }

func (m *Model) SetData(repository provider.Repository, branch string, pull provider.PullRequest, checks provider.ChecksSnapshot) {
	m.Repository, m.Branch, m.Pull, m.Checks, m.Ready, m.Error = repository, branch, pull, checks, true, ""
}

func (m *Model) SetPullRequests(pulls []provider.PullRequest) {
	m.Pulls = append([]provider.PullRequest(nil), pulls...)
}

func (m *Model) SetIssues(issues []provider.Issue) {
	m.Issues = append([]provider.Issue(nil), issues...)
}

func (m *Model) SetReleases(releases []provider.Release) {
	m.Releases = append([]provider.Release(nil), releases...)
}

func (m *Model) SetDetail(detail provider.PullRequestDetail) {
	m.Detail = &detail
	m.Pull = detail.PullRequest
	m.Ready = true
	if len(detail.Files) > 0 || len(detail.Commits) > 0 {
		m.Error = ""
	}
}

func (m *Model) SetComments(comments []provider.ReviewComment) {
	m.Comments = append([]provider.ReviewComment(nil), comments...)
}

func (m *Model) SelectRun(delta int) {
	if len(m.Checks.Runs) == 0 {
		m.SelectedRun = 0
		return
	}
	m.SelectedRun += delta
	if m.SelectedRun < 0 {
		m.SelectedRun = len(m.Checks.Runs) - 1
	}
	if m.SelectedRun >= len(m.Checks.Runs) {
		m.SelectedRun = 0
	}
}

func (m *Model) SetError(repository provider.Repository, branch string, err error) {
	m.Repository, m.Branch, m.Ready = repository, branch, false
	if err == nil {
		m.Error = "provider unavailable"
	} else {
		m.Error = platform.SafeText(err.Error())
	}
}

func (m Model) View() string {
	lines := []string{"GitHub"}
	if m.Repository.Owner == "" || m.Repository.Name == "" {
		return strings.Join(append(lines, "  No GitHub repository detected"), "\n")
	}
	lines = append(lines, fmt.Sprintf("Repository: %s/%s", platform.SafeText(m.Repository.Owner), platform.SafeText(m.Repository.Name)), "Branch: "+platform.SafeText(m.Branch))
	if m.Error != "" {
		return strings.Join(append(lines, "  "+m.Error), "\n")
	}
	if !m.Ready {
		return strings.Join(append(lines, "  Loading provider data…"), "\n")
	}
	if len(m.Pulls) > 0 {
		lines = append(lines, fmt.Sprintf("Open pull requests: %d", len(m.Pulls)))
		for _, pull := range m.Pulls {
			lines = append(lines, fmt.Sprintf("  PR #%d: %s [%s]", pull.Number, platform.SafeText(pull.Title), platform.SafeText(pull.State)))
		}
	}
	if len(m.Issues) > 0 {
		lines = append(lines, fmt.Sprintf("Open issues: %d", len(m.Issues)))
		for _, issue := range m.Issues {
			lines = append(lines, fmt.Sprintf("  Issue #%d: %s [%s]", issue.Number, platform.SafeText(issue.Title), platform.SafeText(issue.State)))
		}
	}
	if len(m.Releases) > 0 {
		lines = append(lines, fmt.Sprintf("Releases: %d", len(m.Releases)))
		for _, release := range m.Releases {
			kind := "release"
			if release.Draft {
				kind = "draft"
			} else if release.Prerelease {
				kind = "pre-release"
			}
			lines = append(lines, fmt.Sprintf("  %s: %s [%s]", kind, platform.SafeText(release.TagName), platform.SafeText(release.Name)))
		}
	}
	lines = append(lines, fmt.Sprintf("PR #%d: %s [%s]", m.Pull.Number, platform.SafeText(m.Pull.Title), platform.SafeText(m.Pull.State)))
	if m.Pull.Draft {
		lines = append(lines, "  Draft")
	}
	lines = append(lines, fmt.Sprintf("  %s -> %s  mergeable=%s reviews=%d comments=%d", platform.SafeText(m.Pull.Head), platform.SafeText(m.Pull.Base), platform.SafeText(m.Pull.Mergeable), m.Pull.Reviews, m.Pull.Comments), "Review: "+platform.SafeText(m.Pull.ReviewState), fmt.Sprintf("Checks: %d passing  %d failing  %d pending", m.Checks.Passing, m.Checks.Failing, m.Checks.Pending))
	for i, run := range m.Checks.Runs {
		selected := " "
		if i == m.SelectedRun {
			selected = ">"
		}
		marker := "✓"
		if run.Status != "completed" {
			marker = "…"
		} else if run.Conclusion != "success" && run.Conclusion != "neutral" && run.Conclusion != "skipped" {
			marker = "!"
		}
		lines = append(lines, fmt.Sprintf(" %s%s check %s [%s]", selected, marker, platform.SafeText(run.Name), platform.SafeText(run.Conclusion)))
		if run.Failure != "" {
			lines = append(lines, "    "+platform.SafeText(run.Failure))
		}
		if run.URL != "" {
			lines = append(lines, "    "+platform.SafeText(run.URL))
		}
	}
	if m.Pull.URL != "" {
		lines = append(lines, "URL: "+platform.SafeText(m.Pull.URL))
	}
	if m.Detail != nil {
		lines = append(lines, fmt.Sprintf("Commits: %d  Files: %d", len(m.Detail.Commits), len(m.Detail.Files)))
		for _, file := range m.Detail.Files {
			lines = append(lines, fmt.Sprintf("  %s %s +%d -%d", platform.SafeText(file.Status), platform.SafeText(file.Path), file.Additions, file.Deletions))
		}
	}
	if len(m.Comments) > 0 {
		lines = append(lines, fmt.Sprintf("Review comments: %d", len(m.Comments)))
		for _, comment := range m.Comments {
			location := comment.Path
			if comment.Line > 0 {
				location = fmt.Sprintf("%s:%d", location, comment.Line)
			}
			lines = append(lines, "  @"+platform.SafeText(comment.Author)+" "+platform.SafeText(location)+": "+platform.SafeText(comment.Body))
		}
	}
	return strings.Join(lines, "\n")
}
