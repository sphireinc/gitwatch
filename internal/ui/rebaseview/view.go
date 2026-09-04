// Package rebaseview renders the pure interactive-rebase planning workspace.
// It owns selection and base-choice state; Git execution remains outside it.
package rebaseview

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/rebase"
)

// Base is an explicit rebase base choice. Ref is never inferred by the view.
type Base struct {
	Label string
	Ref   string
}

// Commit is the presentation metadata for one historical commit.
type Commit struct {
	SHA, Author, Subject, Signature string
	Unix                            int64
}

// Model is the state of the planning workspace.
type Model struct {
	Branch, Upstream string
	Base             Base
	Bases            []Base
	Commits          []Commit
	Plan             rebase.Plan
	Selected         int
	BaseSelected     int
	Marked           map[int]bool
	BaseMode         bool
	Loading          bool
	Error            string
	Published        bool
	ReachableRemote  bool
}

// MouseAction identifies a rebase workspace click.
type MouseAction uint8

const (
	MouseNone MouseAction = iota
	MouseChooseBase
	MouseSelectPlan
	MouseStart
	MouseCancel
)

// New constructs an explicit planning workspace from history metadata.
func New(branch, upstream string, choices []Base, commits []history.Commit) (Model, error) {
	m := Model{Branch: branch, Upstream: upstream, Bases: append([]Base(nil), choices...)}
	if len(m.Bases) > 0 {
		m.Base = m.Bases[0]
	}
	entries := make([]string, 0, len(commits))
	presentation := make([]Commit, 0, len(commits))
	for index := len(commits) - 1; index >= 0; index-- {
		commit := commits[index]
		entries = append(entries, "pick "+commit.SHA+" "+commit.Subject)
		presentation = append(presentation, Commit{SHA: commit.SHA, Author: commit.Author, Subject: commit.Subject, Signature: commit.Signature, Unix: commit.Unix})
	}
	plan, err := rebase.Parse(strings.Join(entries, "\n") + trailingNewline(len(entries) > 0))
	if err != nil {
		return Model{}, err
	}
	m.Commits, m.Plan = presentation, plan
	m.Marked = make(map[int]bool)
	return m, nil
}

func trailingNewline(nonEmpty bool) string {
	if nonEmpty {
		return "\n"
	}
	return ""
}

// SetBase selects an explicit base and never mutates the plan.
func (m *Model) SetBase(index int) error {
	if index < 0 || index >= len(m.Bases) {
		return fmt.Errorf("rebase base index %d is out of bounds", index)
	}
	m.BaseSelected, m.Base = index, m.Bases[index]
	return nil
}

// StartEnabled reports whether the workspace can submit its current plan.
func (m Model) StartEnabled() bool {
	return !m.Loading && m.Error == "" && m.Base.Ref != "" && m.Plan.Validate() == nil && len(m.Commits) > 0
}

// Move changes the selected plan row or base choice.
func (m *Model) Move(delta int) {
	if m.BaseMode {
		m.BaseSelected = clamp(m.BaseSelected+delta, 0, len(m.Bases)-1)
		return
	}
	m.Selected = clamp(m.Selected+delta, 0, len(m.Plan.Entries())-1)
}

// ToggleMark adds or removes the selected plan row from the action range.
func (m *Model) ToggleMark() {
	if m.Marked == nil {
		m.Marked = make(map[int]bool)
	}
	if m.Marked[m.Selected] {
		delete(m.Marked, m.Selected)
	} else {
		m.Marked[m.Selected] = true
	}
}

// ApplyAction changes the selected or marked commit entries. Drop and
// published-history rewrites require an explicit confirmation value.
func (m *Model) ApplyAction(action rebase.Action, confirm bool) error {
	indexes := m.selectedIndexes()
	if (m.Published || m.ReachableRemote) && action != rebase.Pick && !confirm {
		return fmt.Errorf("rewriting published or remote-reachable commits requires confirmation")
	}
	plan, err := m.Plan.ChangeActions(indexes, action)
	if err != nil {
		m.Error = err.Error()
		return err
	}
	m.Plan, m.Error = plan, ""
	return nil
}

// MoveSelection moves the selected contiguous range by one record while
// preserving its relative order and selection.
func (m *Model) MoveSelection(delta int) error {
	indexes := m.selectedIndexes()
	if len(indexes) == 0 {
		return nil
	}
	start, end := indexes[0], indexes[0]
	for _, index := range indexes[1:] {
		if index != end+1 {
			return fmt.Errorf("selected range must be contiguous")
		}
		end = index
	}
	if delta < 0 {
		if start == 0 {
			return nil
		}
		if err := m.moveRange(start, end, start-1); err != nil {
			return err
		}
		m.shiftSelection(-1)
		return nil
	}
	if delta > 0 {
		if end == len(m.Plan.Entries())-1 {
			return nil
		}
		if err := m.moveRange(start, end, end+2); err != nil {
			return err
		}
		m.shiftSelection(1)
	}
	return nil
}

func (m *Model) moveRange(start, end, destination int) error {
	plan, err := m.Plan.MoveRange(start, end, destination)
	if err != nil {
		m.Error = err.Error()
		return err
	}
	m.Plan, m.Error = plan, ""
	return nil
}

func (m Model) selectedIndexes() []int {
	if len(m.Marked) == 0 {
		return []int{m.Selected}
	}
	indexes := make([]int, 0, len(m.Marked))
	for index, marked := range m.Marked {
		if marked {
			indexes = append(indexes, index)
		}
	}
	if len(indexes) == 0 {
		return []int{m.Selected}
	}
	sort.Ints(indexes)
	return indexes
}

func (m *Model) shiftSelection(delta int) {
	if len(m.Marked) == 0 {
		m.Selected += delta
		return
	}
	marked := make(map[int]bool, len(m.Marked))
	for index := range m.Marked {
		marked[index+delta] = true
	}
	m.Marked = marked
	m.Selected += delta
}

// View renders a terminal-safe planning view at any width.
func (m Model) View() string {
	lines := []string{"Interactive rebase", "", "Branch: " + platform.SafeText(m.Branch)}
	base := m.Base.Label
	if base == "" {
		base = "(choose an explicit base)"
	}
	lines = append(lines, "Base: "+platform.SafeText(base))
	if m.Upstream != "" {
		lines = append(lines, "Upstream: "+platform.SafeText(m.Upstream))
	}
	lines = append(lines, fmt.Sprintf("Commits: %d", len(m.Commits)))
	if m.Published || m.ReachableRemote {
		lines = append(lines, "WARNING: selected history is published or reachable from a remote-tracking ref")
	}
	if m.BaseMode {
		lines = append(lines, "", "Choose base:")
		for index, choice := range m.Bases {
			prefix := "  "
			if index == m.BaseSelected {
				prefix = "> "
			}
			lines = append(lines, prefix+platform.SafeText(choice.Label)+" ("+platform.SafeText(choice.Ref)+")")
		}
	} else {
		lines = append(lines, "", "Plan:")
		entries := m.Plan.Entries()
		for index, entry := range entries {
			prefix := "  "
			if index == m.Selected {
				prefix = "> "
			}
			if m.Marked[index] {
				prefix = "* "
			}
			if entry.Kind() == rebase.CommitEntry {
				target := ""
				if entry.Action() == rebase.Squash || entry.Action() == rebase.Fixup {
					if targetIndex, err := m.Plan.SquashTarget(index); err == nil {
						target = " -> " + platform.SafeText(m.Plan.Entries()[targetIndex].SHA())
					}
				}
				lines = append(lines, prefix+string(entry.Action())+" "+platform.SafeText(entry.SHA())+" "+platform.SafeText(entry.Subject())+target)
			} else {
				lines = append(lines, prefix+platform.SafeText(entry.Raw()))
			}
		}
	}
	if m.Error != "" {
		lines = append(lines, "", "ERROR: "+platform.SafeText(m.Error))
	}
	start := "disabled"
	if m.StartEnabled() {
		start = "ready"
	}
	lines = append(lines, "", "[Cancel]  Start: "+start+"  [p] pick [s] squash [f] fixup [r] reword [e] edit [d] drop  [space] mark  [</>] move")
	return strings.Join(lines, "\n")
}

// Click maps a local view coordinate to the same actions exposed by the
// keyboard controls. Coordinates are zero-based within Model.View.
func (m Model) Click(x, y, width, height int) (MouseAction, int) {
	if y < 0 || x < 0 {
		return MouseNone, -1
	}
	if height > 0 && y == height-1 {
		if x < width/2 {
			return MouseCancel, -1
		}
		return MouseStart, -1
	}
	if y == 3 {
		return MouseChooseBase, -1
	}
	if m.BaseMode {
		baseStart := 9
		if m.Upstream == "" {
			baseStart--
		}
		if m.Published || m.ReachableRemote {
			baseStart++
		}
		index := y - baseStart
		if index >= 0 && index < len(m.Bases) {
			return MouseChooseBase, index
		}
		return MouseNone, -1
	}
	planStart := 8
	if m.Upstream != "" {
		planStart++
	}
	if m.Published || m.ReachableRemote {
		planStart++
	}
	index := y - planStart
	if index >= 0 && index < len(m.Plan.Entries()) {
		return MouseSelectPlan, index
	}
	return MouseNone, -1
}

func clamp(value, minimum, maximum int) int {
	if maximum < minimum {
		return 0
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}
