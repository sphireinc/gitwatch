package worktreeview

import (
	"fmt"
	"strings"

	"github.com/jusanchez/gitwatch/internal/worktrees"
)

type Model struct {
	Entries  []worktrees.Entry
	Selected int
}

func New(entries []worktrees.Entry) Model {
	return Model{Entries: append([]worktrees.Entry(nil), entries...)}
}

func (m *Model) SetEntries(entries []worktrees.Entry) {
	selectedPath := ""
	if m.Selected >= 0 && m.Selected < len(m.Entries) {
		selectedPath = m.Entries[m.Selected].Path
	}
	m.Entries = append([]worktrees.Entry(nil), entries...)
	m.Selected = 0
	for i, entry := range m.Entries {
		if entry.Path == selectedPath {
			m.Selected = i
			break
		}
	}
}

func (m *Model) Move(delta int) {
	if len(m.Entries) == 0 {
		m.Selected = 0
		return
	}
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Entries) {
		m.Selected = len(m.Entries) - 1
	}
}

func (m Model) View() string {
	lines := []string{"Worktrees"}
	for i, entry := range m.Entries {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		branch := entry.Branch
		if branch == "" {
			branch = "(detached)"
		}
		states := make([]string, 0, 3)
		if entry.Bare {
			states = append(states, "bare")
		}
		if entry.Locked {
			states = append(states, "locked")
		}
		if entry.Prunable {
			states = append(states, "prunable")
		}
		state := ""
		if len(states) > 0 {
			state = " [" + strings.Join(states, ", ") + "]"
		}
		lines = append(lines, fmt.Sprintf("%s%s · %s%s", prefix, branch, entry.Path, state))
		lines = append(lines, "    HEAD: "+entry.HEAD)
		if entry.LockNote != "" {
			lines = append(lines, "    lock: "+entry.LockNote)
		}
	}
	if len(m.Entries) == 0 {
		lines = append(lines, "  No worktrees")
	}
	return strings.Join(lines, "\n")
}
