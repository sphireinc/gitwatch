package branchview

import (
	"fmt"
	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/platform"
	"strings"
)

type Model struct {
	Entries  []branches.Branch
	Selected int
}

func New(e []branches.Branch) Model { return Model{Entries: e} }
func (m *Model) Move(d int) {
	m.Selected += d
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Entries) {
		m.Selected = len(m.Entries) - 1
	}
}
func (m Model) View() string {
	lines := []string{"Branches"}
	for i, e := range m.Entries {
		p := "  "
		if i == m.Selected {
			p = "> "
		}
		current := ""
		if e.Current {
			current = " *"
		}
		lines = append(lines, fmt.Sprintf("%s%s%s", p, e.Name, current))
		if e.OccupiedPath != "" {
			lines[len(lines)-1] += " · worktree: " + platform.SafeText(e.OccupiedPath)
		}
		if e.Upstream != "" {
			lines[len(lines)-1] += fmt.Sprintf(" · ahead %d/behind %d", e.Ahead, e.Behind)
		}
	}
	return strings.Join(lines, "\n")
}
