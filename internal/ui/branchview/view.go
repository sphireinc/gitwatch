package branchview

import (
	"fmt"
	"github.com/jusanchez/gitwatch/internal/branches"
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
	}
	return strings.Join(lines, "\n")
}
