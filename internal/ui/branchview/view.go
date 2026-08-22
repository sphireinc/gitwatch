// Package branchview renders branch rows with filtering and sorting controls.
package branchview

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/branches"
	"github.com/sphireinc/git-watch/internal/platform"
)

// SortKey identifies the branch field used for ordering.
type SortKey string

const (
	// SortName orders branches by name.
	SortName SortKey = "name"
	// SortAhead orders branches by ahead count.
	SortAhead SortKey = "ahead"
	// SortBehind orders branches by behind count.
	SortBehind SortKey = "behind"
	// SortLastCommit orders branches by commit timestamp.
	SortLastCommit SortKey = "last-commit"
)

// Model stores filtered, sorted branch rows and the current selection.
type Model struct {
	Entries    []branches.Branch
	AllEntries []branches.Branch
	Selected   int
	Query      string
	Sort       SortKey
	Desc       bool
}

// New creates a branch view model from the supplied rows.
func New(e []branches.Branch) Model {
	m := Model{AllEntries: append([]branches.Branch(nil), e...), Sort: SortName}
	m.apply()
	return m
}

// SetEntries replaces branch rows while preserving the selected branch.
func (m *Model) SetEntries(entries []branches.Branch) {
	selected := ""
	if m.Selected >= 0 && m.Selected < len(m.Entries) {
		selected = m.Entries[m.Selected].Name
	}
	m.AllEntries = append([]branches.Branch(nil), entries...)
	m.apply()
	m.Selected = 0
	for i, entry := range m.Entries {
		if entry.Name == selected {
			m.Selected = i
			break
		}
	}
}

// SetFilter applies a case-insensitive name or upstream filter.
func (m *Model) SetFilter(query string) {
	m.Query = query
	m.apply()
	m.Selected = 0
}

// CycleSort advances through supported sort fields and direction.
func (m *Model) CycleSort() SortKey {
	keys := []SortKey{SortName, SortAhead, SortBehind, SortLastCommit}
	index := 0
	for i, key := range keys {
		if key == m.Sort {
			index = i
			break
		}
	}
	if index == len(keys)-1 {
		m.Desc = !m.Desc
		if !m.Desc {
			index = 0
		}
	} else {
		index++
	}
	m.Sort = keys[index]
	m.apply()
	m.Selected = 0
	return m.Sort
}

func (m *Model) apply() {
	query := strings.ToLower(strings.TrimSpace(m.Query))
	m.Entries = m.Entries[:0]
	for _, entry := range m.AllEntries {
		if query == "" || strings.Contains(strings.ToLower(entry.Name), query) || strings.Contains(strings.ToLower(entry.Upstream), query) {
			m.Entries = append(m.Entries, entry)
		}
	}
	sort.SliceStable(m.Entries, func(i, j int) bool {
		left, right := m.Entries[i], m.Entries[j]
		var less bool
		switch m.Sort {
		case SortAhead:
			less = left.Ahead < right.Ahead
		case SortBehind:
			less = left.Behind < right.Behind
		case SortLastCommit:
			less = left.LastCommitUnix < right.LastCommitUnix
		default:
			less = strings.ToLower(left.Name) < strings.ToLower(right.Name)
		}
		if m.Desc {
			return !less && left.Name != right.Name
		}
		return less
	})
}

// SortLabel returns the current sort field name.
func (m Model) SortLabel() string { return string(m.Sort) }

// Move shifts the selected branch and clamps it to the filtered rows.
func (m *Model) Move(d int) {
	m.Selected += d
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Entries) {
		m.Selected = len(m.Entries) - 1
	}
}

// View renders the filtered branch list and synchronization metadata.
func (m Model) View() string {
	header := "Branches"
	if m.Query != "" {
		header += " · filter: " + platform.SafeText(m.Query)
	}
	header += " · sort: " + m.SortLabel()
	lines := []string{header}
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
		if e.Merged {
			lines[len(lines)-1] += " · merged"
		}
		if e.LastCommitUnix != 0 {
			lines[len(lines)-1] += " · last commit " + time.Unix(e.LastCommitUnix, 0).Format("2006-01-02")
		}
	}
	return strings.Join(lines, "\n")
}
