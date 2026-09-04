// Package rebaseview renders the pure interactive-rebase planning workspace.
// It owns selection and base-choice state; Git execution remains outside it.
package rebaseview

import (
	"fmt"
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
			if entry.Kind() == rebase.CommitEntry {
				lines = append(lines, prefix+string(entry.Action())+" "+platform.SafeText(entry.SHA())+" "+platform.SafeText(entry.Subject()))
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
	lines = append(lines, "", "[Cancel]  Start: "+start+"  [b] choose base  [j/k] move  [enter] select/start")
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
