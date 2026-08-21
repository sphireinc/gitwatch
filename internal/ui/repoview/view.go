package repoview

import (
	"fmt"
	"strings"

	"github.com/jusanchez/gitwatch/internal/platform"
	"github.com/jusanchez/gitwatch/internal/registry"
)

type Model struct {
	Rows     []registry.Row
	Selected int
}

func New(rows []registry.Row) Model { return Model{Rows: append([]registry.Row(nil), rows...)} }

func (m *Model) SetRows(rows []registry.Row) {
	selected := ""
	if m.Selected >= 0 && m.Selected < len(m.Rows) {
		selected = m.Rows[m.Selected].Repository.Path
	}
	m.Rows = append([]registry.Row(nil), rows...)
	m.Selected = 0
	for i, row := range m.Rows {
		if row.Repository.Path == selected {
			m.Selected = i
			break
		}
	}
}

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
	lines := []string{"Repositories"}
	for i, row := range m.Rows {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		state := row.State
		lines = append(lines, fmt.Sprintf("%s%s · %s [%s] dirty:%d +%d/-%d stashes:%d remotes:%d", prefix, platform.SafeText(row.Repository.Name), platform.SafeText(row.Branch), state, row.Dirty, row.Ahead, row.Behind, row.Stashes, row.Remotes))
		lines = append(lines, "    "+platform.SafeText(row.Repository.Path))
	}
	if len(m.Rows) == 0 {
		lines = append(lines, "  No repositories")
	}
	return strings.Join(lines, "\n")
}
