package githubview

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/provider"
)

type ResourceWarning struct {
	Resource string
	Message  string
}

type Model struct {
	Repository      provider.Repository
	Branch          string
	Pull            provider.PullRequest
	Pulls           []provider.PullRequest
	Issues          []provider.Issue
	Releases        []provider.Release
	SelectedIssue   int
	SelectedRelease int
	Detail          *provider.PullRequestDetail
	Comments        []provider.ReviewComment
	SelectedComment int
	Checks          provider.ChecksSnapshot
	Workflows       []provider.WorkflowRun
	SelectedRun     int
	Ready           bool
	Error           string
	ErrorHint       string
	State           provider.State
	RetryAfter      string
	ProviderStale   bool
	Warnings        []ResourceWarning
	workflowHeight  int
	ReviewFocused   bool
	ReviewOffset    int
}

func New() Model { return Model{} }

func (m *Model) SetData(repository provider.Repository, branch string, pull provider.PullRequest, checks provider.ChecksSnapshot) {
	if m.Repository != repository || m.Branch != branch || m.Pull.Number != pull.Number {
		m.Detail = nil
		m.Comments = nil
		m.SelectedComment = 0
		m.ReviewFocused, m.ReviewOffset = false, 0
	}
	m.Repository, m.Branch, m.Pull, m.Checks, m.Ready, m.Error = repository, branch, pull, checks, true, ""
	m.State, m.RetryAfter = provider.StateAvailable, ""
	m.Warnings = nil
}

func (m *Model) SetPullRequests(pulls []provider.PullRequest) {
	m.Pulls = append([]provider.PullRequest(nil), pulls...)
}

// SetWorkflows preserves a selected workflow by provider identity on refresh.
func (m *Model) SetWorkflows(runs []provider.WorkflowRun) {
	var selectedID int64
	if m.SelectedRun >= 0 && m.SelectedRun < len(m.Workflows) {
		selectedID = m.Workflows[m.SelectedRun].ID
	}
	m.Workflows = append([]provider.WorkflowRun(nil), runs...)
	m.SelectedRun = 0
	for i, run := range m.Workflows {
		if run.ID == selectedID {
			m.SelectedRun = i
			break
		}
	}
}

func (m *Model) SetIssues(issues []provider.Issue) {
	m.Issues = append([]provider.Issue(nil), issues...)
	if m.SelectedIssue >= len(m.Issues) {
		m.SelectedIssue = max(0, len(m.Issues)-1)
	}
}

func (m *Model) SetReleases(releases []provider.Release) {
	m.Releases = append([]provider.Release(nil), releases...)
	if m.SelectedRelease >= len(m.Releases) {
		m.SelectedRelease = max(0, len(m.Releases)-1)
	}
}

// SetProviderFreshness records whether any provider collection shown in the
// workspace came from stale cache after a refresh failure. Local Git state is
// intentionally independent of this optional provider signal.
func (m *Model) SetProviderFreshness(stale bool) { m.ProviderStale = stale }

func (m *Model) SetWarnings(warnings []ResourceWarning) {
	const maxWarnings = 8
	if len(warnings) > maxWarnings {
		warnings = warnings[:maxWarnings]
	}
	m.Warnings = append(m.Warnings[:0], warnings...)
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
	if m.SelectedComment >= len(m.Comments) {
		m.SelectedComment = max(0, len(m.Comments)-1)
	}
}

func (m *Model) SelectComment(delta int) {
	m.ReviewFocused, m.ReviewOffset = true, 0
	if len(m.Comments) == 0 {
		m.SelectedComment = 0
		return
	}
	m.SelectedComment = (m.SelectedComment + delta) % len(m.Comments)
	if m.SelectedComment < 0 {
		m.SelectedComment += len(m.Comments)
	}
}

func (m *Model) SelectRun(delta int) {
	m.ReviewFocused = false
	count := len(m.Workflows)
	if count == 0 {
		count = len(m.Checks.Runs)
	}
	if count == 0 {
		m.SelectedRun = 0
		return
	}
	m.SelectedRun += delta
	if m.SelectedRun < 0 {
		m.SelectedRun = count - 1
	}
	if m.SelectedRun >= count {
		m.SelectedRun = 0
	}
}

func (m Model) workflowWindow(height int) (start, count int) {
	count = len(m.Workflows)
	if height > 0 {
		count = min(count, max(0, (height-5)/2))
	}
	if count == 0 {
		return 0, 0
	}
	selected := min(max(0, m.SelectedRun), len(m.Workflows)-1)
	start = min(max(0, selected-count/2), len(m.Workflows)-count)
	return start, count
}

// SelectWorkflowRow accepts a content-relative terminal row, not a provider ID.
func (m *Model) SelectWorkflowRow(row, height int) bool {
	if m.ReviewFocused {
		return false
	}
	if m.Error != "" || !m.Ready || m.Repository.Owner == "" || row < 5 {
		return false
	}
	start, count := m.workflowWindow(height)
	index := (row - 5) / 2
	if index >= count {
		return false
	}
	m.SelectedRun = start + index
	return true
}

// ViewWithSize keeps selected workflow rows visible without doing provider I/O.
func (m Model) ViewWithSize(width, height int) string {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	m.workflowHeight = height
	var lines []string
	if m.ReviewFocused && m.Ready && m.Error == "" && m.Repository.Owner != "" {
		lines = m.reviewViewportLines(width, height)
	} else {
		lines = strings.Split(m.View(), "\n")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	for i := range lines {
		lines[i] = runewidth.Truncate(platform.SafeText(lines[i]), width, "…")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) SetError(repository provider.Repository, branch string, err error) {
	m.Repository, m.Branch, m.Ready = repository, branch, false
	m.State = provider.Classify(context.Background(), err)
	m.RetryAfter = ""
	var httpErr *provider.HTTPError
	if errors.As(err, &httpErr) {
		m.RetryAfter = platform.SafeText(httpErr.RetryAfter)
	}
	if err == nil {
		m.Error = "provider unavailable"
	} else {
		m.Error = platform.SafeText(err.Error())
	}
	switch m.State {
	case provider.StateNotConfigured:
		if errors.Is(err, provider.ErrNoGitHubRemote) {
			m.ErrorHint = "Add a GitHub remote to use this optional workspace."
		} else {
			m.ErrorHint = "Configure a GitHub token or sign in with gh auth login."
		}
	case provider.StateUnauthorized:
		m.ErrorHint = "The token is missing permission for this repository or action; local Git remains authoritative."
	case provider.StateRateLimited:
		m.ErrorHint = "Wait for the provider quota window before retrying."
	case provider.StateUnavailable:
		m.ErrorHint = "Check network connectivity and retry; local Git remains authoritative."
	default:
		m.ErrorHint = "Retry provider loading; local Git remains authoritative."
	}
}

func (m Model) View() string {
	lines := []string{"GitHub"}
	if m.Repository.Owner == "" || m.Repository.Name == "" {
		if m.Error == "" {
			return strings.Join(append(lines, "  No GitHub repository detected"), "\n")
		}
		lines = append(lines, "  No GitHub repository detected", "  provider state: "+platform.SafeText(string(m.State)), "  "+platform.SafeText(m.Error))
		if m.ErrorHint != "" {
			lines = append(lines, "  Hint: "+platform.SafeText(m.ErrorHint))
		}
		return strings.Join(lines, "\n")
	}
	lines = append(lines, fmt.Sprintf("Repository: %s/%s", platform.SafeText(m.Repository.Owner), platform.SafeText(m.Repository.Name)), "Branch: "+platform.SafeText(m.Branch))
	if m.Error != "" {
		status := "  provider state: " + platform.SafeText(string(m.State))
		if m.RetryAfter != "" {
			status += " (retry after " + m.RetryAfter + ")"
		}
		lines = append(lines, status, "  "+m.Error)
		if m.ErrorHint != "" {
			lines = append(lines, "  Hint: "+platform.SafeText(m.ErrorHint))
		}
		return strings.Join(lines, "\n")
	}
	if !m.Ready {
		return strings.Join(append(lines, "  Loading provider data…"), "\n")
	}
	cacheState := "fresh"
	if m.ProviderStale {
		cacheState = "stale"
	}
	lines = append(lines, "  provider cache: "+cacheState)
	if len(m.Workflows) > 0 {
		lines = append(lines, fmt.Sprintf("Workflow runs: %d (j/k select, W logs, ! rerun failed, K cancel)", len(m.Workflows)))
		start, count := m.workflowWindow(m.workflowHeight)
		now := time.Now()
		for i := start; i < start+count; i++ {
			run := m.Workflows[i]
			prefix := "  "
			if i == m.SelectedRun {
				prefix = "> "
			}
			lines = append(lines,
				fmt.Sprintf("%s%s [%s/%s] attempt:%d elapsed:%s", prefix, workflowText(run.Name), workflowText(run.Status), workflowText(run.Conclusion), run.Attempt, run.Elapsed(now).Truncate(time.Second)),
				"    "+workflowText(run.URL))
		}
	}
	if len(m.Warnings) > 0 {
		lines = append(lines, "Provider warnings:")
		for _, warning := range m.Warnings {
			lines = append(lines, "  "+platform.SafeText(warning.Resource)+": "+platform.SafeText(warning.Message))
		}
	}
	if len(m.Pulls) > 0 {
		lines = append(lines, fmt.Sprintf("Open pull requests: %d", len(m.Pulls)))
		for _, pull := range m.Pulls {
			lines = append(lines, fmt.Sprintf("  PR #%d: %s [%s]", pull.Number, platform.SafeText(pull.Title), platform.SafeText(pull.State)))
		}
	}
	if len(m.Issues) > 0 {
		lines = append(lines, fmt.Sprintf("Open issues: %d", len(m.Issues)))
		for index, issue := range m.Issues {
			prefix := "  "
			if index == m.SelectedIssue {
				prefix = "> "
			}
			lines = append(lines, fmt.Sprintf("%sIssue #%d: %s [%s]", prefix, issue.Number, platform.SafeText(issue.Title), platform.SafeText(issue.State)))
		}
	}
	if len(m.Releases) > 0 {
		lines = append(lines, fmt.Sprintf("Releases: %d", len(m.Releases)))
		for index, release := range m.Releases {
			prefix := "  "
			if index == m.SelectedRelease {
				prefix = "> "
			}
			kind := "release"
			if release.Draft {
				kind = "draft"
			} else if release.Prerelease {
				kind = "pre-release"
			}
			lines = append(lines, fmt.Sprintf("%s%s: %s [%s]", prefix, kind, platform.SafeText(release.TagName), platform.SafeText(release.Name)))
		}
	}
	if m.Pull.Number == 0 {
		lines = append(lines, "No open pull request for the current branch")
	} else {
		lines = append(lines, fmt.Sprintf("PR #%d: %s [%s]", m.Pull.Number, platform.SafeText(m.Pull.Title), platform.SafeText(m.Pull.State)))
		if m.Pull.Draft {
			lines = append(lines, "  Draft")
		}
		lines = append(lines, fmt.Sprintf("  %s -> %s  mergeable=%s reviews=%d comments=%d", platform.SafeText(m.Pull.Head), platform.SafeText(m.Pull.Base), platform.SafeText(m.Pull.Mergeable), m.Pull.Reviews, m.Pull.Comments), "Review: "+platform.SafeText(m.Pull.ReviewState))
	}
	lines = append(lines, fmt.Sprintf("Checks: %d passing  %d failing  %d pending", m.Checks.Passing, m.Checks.Failing, m.Checks.Pending))
	for i, run := range m.Checks.Runs {
		selected := " "
		if len(m.Workflows) == 0 && i == m.SelectedRun {
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
	if m.Pull.Number > 0 && m.Pull.URL != "" {
		lines = append(lines, "URL: "+platform.SafeText(m.Pull.URL))
	}
	if m.Pull.Number > 0 && m.Detail != nil {
		lines = append(lines, fmt.Sprintf("Commits: %d  Files: %d", len(m.Detail.Commits), len(m.Detail.Files)))
		for _, file := range m.Detail.Files {
			lines = append(lines, fmt.Sprintf("  %s %s +%d -%d", platform.SafeText(file.Status), platform.SafeText(file.Path), file.Additions, file.Deletions))
		}
	}
	if m.Pull.Number > 0 && len(m.Comments) > 0 {
		lines = append(lines, fmt.Sprintf("Review comments: %d", len(m.Comments)))
		for index, comment := range m.Comments {
			prefix := "  "
			if index == m.SelectedComment {
				prefix = "> "
			}
			location := comment.Path
			if comment.Line > 0 {
				location = fmt.Sprintf("%s:%d", location, comment.Line)
			}
			lines = append(lines, prefix+"@"+platform.SafeText(comment.Author)+" "+platform.SafeText(location)+": "+platform.SafeText(comment.Body))
		}
	}
	return strings.Join(lines, "\n")
}

func workflowText(value string) string {
	return strings.NewReplacer("\n", " ", "\t", " ").Replace(platform.SafeText(value))
}
