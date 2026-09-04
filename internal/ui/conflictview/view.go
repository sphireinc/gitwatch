// Package conflictview contains the pure state and presentation model for
// resolving conflicts. Git mutations are returned as intents for the app's
// repository-scoped operation coordinator.
package conflictview

import (
	"crypto/sha256"
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
	ActionRegionOurs
	ActionRegionTheirs
	ActionRegionBoth
	ActionRegionManual
)

type MouseAction uint8

const (
	MouseNone MouseAction = iota
	MouseSelectConflict
	MouseChooseOurs
	MouseChooseTheirs
	MouseChooseBoth
	MouseMarkResolved
	MouseRestoreUnresolved
	MouseStatus
)

type Model struct {
	Operation sequencer.Kind
	Target    string
	Conflicts []conflicts.Conflict
	Selected  int
	Hunk      int
	Wide      bool
	Detail    *Detail
}

type Content struct {
	Text        string
	Binary      bool
	InvalidUTF8 bool
	Truncated   bool
	Missing     bool
	Hash        [32]byte
	Regions     int
}

type Detail struct {
	Path   []byte
	Ours   Content
	Theirs Content
	Result Content
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
	m.Detail = nil
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

// SetDetail accepts detail only for the currently selected conflict. A stale
// asynchronous load therefore cannot replace a newer selection.
func (m *Model) SetDetail(detail Detail) bool {
	selected, ok := m.SelectedConflict()
	if !ok || string(selected.Path) != string(detail.Path) {
		return false
	}
	detail.Path = append([]byte(nil), detail.Path...)
	m.Detail = &detail
	return true
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
		if m.Detail != nil {
			lines = append(lines, "", "Selected content:")
			lines = append(lines, contentLine("Ours", m.Detail.Ours), contentLine("Theirs", m.Detail.Theirs), contentLine("Result", m.Detail.Result))
		}
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
	lines = append(lines, "", "[j/k] conflict  [n/p] hunk  [o/t/b] whole-file  [O/T/B/M] region  [e] edit  [m] mark  [u] restore  [c] continue  [x] abort  [1] status  [esc] back")
	if height > 0 && len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

func contentLine(label string, content Content) string {
	if content.Missing {
		return label + ": (missing)"
	}
	if content.Binary {
		return fmt.Sprintf("%s: binary content (%s)", label, sizeLabel(content))
	}
	if content.InvalidUTF8 {
		return fmt.Sprintf("%s: invalid UTF-8 (%s)", label, sizeLabel(content))
	}
	if content.Truncated {
		return label + ": content exceeds display limit"
	}
	return label + ": " + platform.SafeText(content.Text)
}

func sizeLabel(content Content) string {
	if content.Truncated {
		return "oversized"
	}
	return "available"
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
	case "O":
		return ActionRegionOurs
	case "T":
		return ActionRegionTheirs
	case "B":
		return ActionRegionBoth
	case "M":
		return ActionRegionManual
	}
	return ActionNone
}

func (m Model) SelectedRegion() (int, [32]byte, bool) {
	if m.Detail == nil || m.Detail.Result.Regions <= 0 || m.Hunk < 0 || m.Hunk >= m.Detail.Result.Regions {
		return 0, sha256.Sum256(nil), false
	}
	return m.Hunk, m.Detail.Result.Hash, true
}

func (m Model) RegionCount() int {
	if m.Detail == nil {
		return 0
	}
	return m.Detail.Result.Regions
}

// Click maps only explicit list rows and footer action zones. Resolution
// actions are never inferred from a generic row click.
func (m Model) Click(x, y, width, height int) (MouseAction, int) {
	if x < 0 || y < 0 || width <= 0 || height <= 0 {
		return MouseNone, -1
	}
	if y == height-1 {
		switch {
		case x < width/6:
			return MouseChooseOurs, -1
		case x < width/3:
			return MouseChooseTheirs, -1
		case x < width/2:
			return MouseChooseBoth, -1
		case x < width*2/3:
			return MouseMarkResolved, -1
		case x < width*5/6:
			return MouseRestoreUnresolved, -1
		default:
			return MouseStatus, -1
		}
	}
	listStart := 9
	index := y - listStart
	if index >= 0 && index < len(m.Conflicts) {
		return MouseSelectConflict, index
	}
	return MouseNone, -1
}
