package historyview

import (
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/platform"
)

// Model owns the small amount of interaction state needed to render history.
// Git loading is deliberately kept outside this package so views remain pure.
type Model struct {
	Rows     []history.GraphRow
	Selected int
	Filter   string
	Pulse    uint8
}

func New(commits []history.Commit) Model { return Model{Rows: history.BuildGraph(commits)} }

func (m *Model) SetCommits(commits []history.Commit) {
	selectedSHA := ""
	if m.Selected >= 0 && m.Selected < len(m.Rows) {
		selectedSHA = m.Rows[m.Selected].Commit.SHA
	}
	m.Rows = history.BuildGraph(history.Filter(commits, m.Filter))
	m.Selected = 0
	for i, row := range m.Rows {
		if row.Commit.SHA == selectedSHA {
			m.Selected = i
			break
		}
	}
}

func (m *Model) SetFilter(filter string, commits []history.Commit) {
	m.Filter = filter
	m.SetCommits(commits)
}

func (m *Model) SetPulse(pulse uint8) { m.Pulse = pulse }

func (m *Model) Move(delta int) {
	if len(m.Rows) == 0 {
		m.Selected = 0
		return
	}
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Rows) {
		m.Selected = len(m.Rows) - 1
	}
}

func (m Model) View() string {
	lines := []string{"History"}
	if m.Filter != "" {
		lines = append(lines, "Filter: "+m.Filter)
	}
	for i, row := range m.Rows {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		lane := "│"
		if row.Lane == 0 {
			lane = "●"
		}
		if i == m.Selected && m.Pulse%2 == 1 {
			lane = "◉"
		}
		refs := append([]string{}, row.Branches...)
		refs = append(refs, row.Tags...)
		for j := range refs {
			refs[j] = platform.SafeText(refs[j])
		}
		decoration := ""
		if row.Head {
			decoration = " HEAD"
		}
		if len(refs) > 0 {
			decoration += " [" + strings.Join(refs, ", ") + "]"
		}
		lines = append(lines, fmt.Sprintf("%s%s %s %s%s", prefix, lane, platform.SafeText(row.Commit.Short), platform.SafeText(row.Commit.Subject), decoration))
	}
	if len(m.Rows) == 0 {
		lines = append(lines, "  No commits")
	}
	return strings.Join(lines, "\n")
}
