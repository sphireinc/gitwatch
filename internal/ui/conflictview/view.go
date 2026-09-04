// Package conflictview contains the pure state and presentation model for
// resolving conflicts. Git mutations are returned as intents for the app's
// repository-scoped operation coordinator.
package conflictview

import (
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/conflicts"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

type Action uint8

const (
	ActionNone Action = iota
	ActionPreviousConflict
	ActionNextConflict
	ActionPreviousHunk
	ActionNextHunk
	ActionChooseOurs
	ActionChooseTheirs
	ActionChooseBoth
	ActionEditExternal
	ActionMarkResolved
	ActionRestoreUnresolved
	ActionContinue
	ActionAbort
	ActionStatus
)

type Model struct {
	Operation sequencer.Kind
	Target    string
	Conflicts []conflicts.Conflict
	Selected  int
	Hunk      int
	Wide      bool
}

func New() Model { return Model{Wide: true, Selected: -1} }

// SetSnapshot replaces all derived view state from the latest authoritative
// snapshot and keeps the selected path when Git still reports it.
func (m *Model) SetSnapshot(operation sequencer.Kind, target string, values []conflicts.Conflict) {
	var selectedPath []byte
	if m.Selected >= 0 && m.Selected < len(m.Conflicts) {
		selectedPath = m.Conflicts[m.Selected].Bytes()
	}
	m.Operation, m.Target = operation, target
	m.Conflicts = append([]conflicts.Conflict(nil), values...)
	for i := range m.Conflicts {
		m.Conflicts[i].Path = append([]byte(nil), m.Conflicts[i].Path...)
	}
	m.Selected = 0
	if len(m.Conflicts) == 0 {
		m.Selected = -1
		return
	}
	for i := range m.Conflicts {
		if string(m.Conflicts[i].Path) == string(selectedPath) {
			m.Selected = i
			break
		}
	}
	if m.Selected < 0 {
		m.Selected = 0
	}
}

func (m *Model) Move(delta int) {
	if len(m.Conflicts) == 0 {
		m.Selected = -1
		return
	}
	m.Selected = (m.Selected + delta + len(m.Conflicts)) % len(m.Conflicts)
	m.Hunk = 0
}

func (m *Model) MoveHunk(delta int, count int) {
	if count <= 0 {
		m.Hunk = 0
		return
	}
	m.Hunk = (m.Hunk + delta + count) % count
}

func (m Model) SelectedConflict() (conflicts.Conflict, bool) {
	if m.Selected < 0 || m.Selected >= len(m.Conflicts) {
		return conflicts.Conflict{}, false
	}
	return m.Conflicts[m.Selected], true
}

func (m Model) ResolvedCount() int {
	count := 0
	for _, conflict := range m.Conflicts {
		if conflict.Resolution == "resolved" {
			count++
		}
	}
	return count
}

func (m Model) View(width, height int) string {
	if width <= 0 {
		width = 80
	}
	lines := []string{
		"Conflict resolver",
		fmt.Sprintf("Operation: %s  Target: %s", m.Operation.String(), platform.SafeText(m.Target)),
		fmt.Sprintf("Conflicts: %d total, %d resolved", len(m.Conflicts), m.ResolvedCount()),
	}
	if selected, ok := m.SelectedConflict(); ok {
		lines = append(lines, "Selected: "+platform.SafeText(string(selected.Path)), "")
		if m.Wide && width >= 100 {
			lines = append(lines, "Ours                 Theirs               Result")
			lines = append(lines, column(selected.Ours.OID), column(selected.Theirs.OID), column(selected.Resolution))
		} else {
			lines = append(lines, "Ours: "+column(selected.Ours.OID), "Theirs: "+column(selected.Theirs.OID), "Result: "+column(selected.Resolution))
		}
		lines = append(lines, "", "Conflict list:")
		for i, conflict := range m.Conflicts {
			prefix := "  "
			if i == m.Selected {
				prefix = "> "
			}
			lines = append(lines, prefix+platform.SafeText(string(conflict.Path))+" ["+column(conflict.Resolution)+"]")
		}
	} else {
		lines = append(lines, "", "No active conflicts.")
	}
	lines = append(lines, "", "[j/k] conflict  [n/p] hunk  [o] ours  [t] theirs  [b] both  [e] edit  [m] mark  [u] restore  [c] continue  [x] abort  [1] status  [esc] back")
	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func column(value string) string {
	if value == "" {
		return "(missing)"
	}
	return platform.SafeText(value)
}

// Key converts a keyboard action to an explicit coordinator intent.
func Key(key string) Action {
	switch key {
	case "j", "down":
		return ActionNextConflict
	case "k", "up":
		return ActionPreviousConflict
	case "n":
		return ActionNextHunk
	case "p":
		return ActionPreviousHunk
	case "o":
		return ActionChooseOurs
	case "t":
		return ActionChooseTheirs
	case "b":
		return ActionChooseBoth
	case "e":
		return ActionEditExternal
	case "m":
		return ActionMarkResolved
	case "u":
		return ActionRestoreUnresolved
	case "c":
		return ActionContinue
	case "x":
		return ActionAbort
	case "1":
		return ActionStatus
	}
	return ActionNone
}
