package remoteview

import (
	"fmt"
	"strings"
	"time"

	"github.com/jusanchez/gitwatch/internal/remotes"
)

// Model is a render-only projection of remote synchronization state.
type Model struct {
	Dashboard remotes.Dashboard
	Selected  int
}

func New(dashboard remotes.Dashboard) Model { return Model{Dashboard: dashboard} }

func (m *Model) SetDashboard(dashboard remotes.Dashboard) {
	selectedName := ""
	if m.Selected >= 0 && m.Selected < len(m.Dashboard.Remotes) {
		selectedName = m.Dashboard.Remotes[m.Selected].Name
	}
	m.Dashboard = dashboard
	m.Selected = 0
	for i, remote := range dashboard.Remotes {
		if remote.Name == selectedName {
			m.Selected = i
			break
		}
	}
}

func (m *Model) Move(delta int) {
	if len(m.Dashboard.Remotes) == 0 {
		m.Selected = 0
		return
	}
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Dashboard.Remotes) {
		m.Selected = len(m.Dashboard.Remotes) - 1
	}
}

func (m Model) View() string {
	d := m.Dashboard
	lines := []string{"Remotes", fmt.Sprintf("Branch: %s  ahead %d  behind %d", nonEmpty(d.CurrentBranch, "detached"), d.Ahead, d.Behind)}
	for i, remote := range d.Remotes {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		state := "reachable"
		if !remote.Reachable {
			state = "unreachable"
		}
		if d.Stale(remote) {
			state += ", stale"
		}
		if remote.Default {
			state += ", default"
		}
		lines = append(lines, fmt.Sprintf("%s%s [%s]", prefix, remote.Name, state))
		lines = append(lines, "    fetch: "+remote.FetchURL, "    push:  "+remote.PushURL)
		if remote.LastError != "" {
			lines = append(lines, "    error: "+remote.LastError)
		}
	}
	if len(d.Remotes) == 0 {
		lines = append(lines, "  No remotes configured")
	}
	if jobs := d.ActiveJobs(); len(jobs) > 0 {
		lines = append(lines, "", fmt.Sprintf("Active jobs: %d", len(jobs)))
		for _, job := range jobs {
			lines = append(lines, fmt.Sprintf("  %s %s (%s)", job.Operation, job.Remote, job.State))
		}
	}
	if activity := d.RecentActivity(3); len(activity) > 0 {
		lines = append(lines, "", "Recent activity:")
		for _, event := range activity {
			marker := "✓"
			if !event.Success {
				marker = "!"
			}
			lines = append(lines, "  "+marker+" "+event.Operation+": "+event.Message)
		}
	}
	return strings.Join(lines, "\n")
}

func nonEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func DefaultStaleAfter() time.Duration { return 24 * time.Hour }
