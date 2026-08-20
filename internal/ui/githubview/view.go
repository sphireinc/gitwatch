package githubview

import (
	"fmt"
	"strings"

	"github.com/jusanchez/gitwatch/internal/platform"
	"github.com/jusanchez/gitwatch/internal/provider"
)

type Model struct {
	Repository provider.Repository
	Branch     string
	Pull       provider.PullRequest
	Checks     provider.ChecksSnapshot
	Ready      bool
	Error      string
}

func New() Model { return Model{} }

func (m *Model) SetData(repository provider.Repository, branch string, pull provider.PullRequest, checks provider.ChecksSnapshot) {
	m.Repository, m.Branch, m.Pull, m.Checks, m.Ready, m.Error = repository, branch, pull, checks, true, ""
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
	lines = append(lines, fmt.Sprintf("PR #%d: %s [%s]", m.Pull.Number, platform.SafeText(m.Pull.Title), platform.SafeText(m.Pull.State)))
	if m.Pull.Draft {
		lines = append(lines, "  Draft")
	}
	lines = append(lines, fmt.Sprintf("  %s -> %s  mergeable=%s reviews=%d comments=%d", platform.SafeText(m.Pull.Head), platform.SafeText(m.Pull.Base), platform.SafeText(m.Pull.Mergeable), m.Pull.Reviews, m.Pull.Comments), "Review: "+platform.SafeText(m.Pull.ReviewState), fmt.Sprintf("Checks: %d passing  %d failing  %d pending", m.Checks.Passing, m.Checks.Failing, m.Checks.Pending))
	if m.Pull.URL != "" {
		lines = append(lines, "URL: "+platform.SafeText(m.Pull.URL))
	}
	return strings.Join(lines, "\n")
}
