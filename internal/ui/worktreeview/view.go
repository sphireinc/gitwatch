// Package worktreeview renders linked worktrees and their selection state.
package worktreeview

import (
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/worktrees"
)

// Model stores the selection state for the worktree view.
type Model struct {
	Entries  []worktrees.Entry
	Selected int
}

// New creates a worktree view model with entries selected at the first row.
func New(entries []worktrees.Entry) Model {
	return Model{Entries: append([]worktrees.Entry(nil), entries...)}
}

// SetEntries replaces rows while keeping the selection within the new bounds.
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

// Move shifts the selected row by delta and clamps it to the available rows.
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

// View renders the current worktree rows.
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
		lines = append(lines, fmt.Sprintf("%s%s · %s%s", prefix, platform.SafeText(branch), platform.SafeText(entry.Path), state))
		lines = append(lines, "    HEAD: "+platform.SafeText(entry.HEAD))
		if entry.LockNote != "" {
			lines = append(lines, "    lock: "+platform.SafeText(entry.LockNote))
		}
	}
	if len(m.Entries) == 0 {
		lines = append(lines, "  No worktrees")
	}
	return strings.Join(lines, "\n")
}
