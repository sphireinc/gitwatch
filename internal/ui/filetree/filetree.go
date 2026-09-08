// Package filetree builds a collapsible presentation index over Git status
// entries. It never owns or filters repository state; callers provide the
// already-filtered entry indexes from the authoritative status model.
package filetree

import (
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/repo"
)

// Row is one visible tree row. Directory rows have EntryIndex == -1.
type Row struct {
	Path       string
	EntryIndex int
	Depth      int
	Directory  bool
	Counts     Counts
}

// Counts is the status summary for a directory and its visible descendants.
type Counts struct {
	Staged, Unstaged, Untracked, Conflicted int
}

// Model is a pure, in-memory presentation index over status entries.
type Model struct {
	Entries  []repo.Entry
	Visible  []int
	Rows     []Row
	Selected int
	Offset   int
	Expanded map[string]bool
}

// New creates an expanded tree. The visible indexes must refer to entries.
func New(entries []repo.Entry, visible []int, selectedPath string) Model {
	m := Model{Expanded: make(map[string]bool)}
	m.SetEntries(entries, visible, selectedPath)
	return m
}

// SetEntries rebuilds the presentation index and preserves expansion and the
// selected path whenever those paths remain available.
func (m *Model) SetEntries(entries []repo.Entry, visible []int, selectedPath string) {
	previousExpanded := m.Expanded
	if previousExpanded == nil {
		previousExpanded = make(map[string]bool)
	}
	m.Entries = append([]repo.Entry(nil), entries...)
	m.Visible = append([]int(nil), visible...)
	m.Expanded = make(map[string]bool, len(previousExpanded))
	for path, expanded := range previousExpanded {
		m.Expanded[path] = expanded
	}
	m.Rows = buildRows(m.Entries, m.Visible, m.Expanded)
	m.Selected = 0
	for index, row := range m.Rows {
		if !row.Directory && row.Path == selectedPath {
			m.Selected = index
			break
		}
	}
	if m.Selected >= len(m.Rows) {
		m.Selected = max(0, len(m.Rows)-1)
	}
	m.Offset = min(m.Offset, max(0, len(m.Rows)-1))
}

// SelectedEntryIndex resolves the selected row to a status entry.
func (m Model) SelectedEntryIndex() (int, bool) {
	if m.Selected < 0 || m.Selected >= len(m.Rows) {
		return 0, false
	}
	row := m.Rows[m.Selected]
	if row.Directory || row.EntryIndex < 0 || row.EntryIndex >= len(m.Entries) {
		return 0, false
	}
	return row.EntryIndex, true
}

// Move changes the selected tree row and keeps it inside the viewport.
func (m *Model) Move(delta, height int) {
	if len(m.Rows) == 0 {
		return
	}
	m.Selected = max(0, min(len(m.Rows)-1, m.Selected+delta))
	m.keepVisible(height)
}

// Page moves by one viewport, preserving the selected row invariant.
func (m *Model) Page(delta, height int) { m.Move(delta*max(1, height), height) }

// Home selects the first row.
func (m *Model) Home(height int) { m.Selected, m.Offset = 0, 0; m.keepVisible(height) }

// End selects the last row.
func (m *Model) End(height int) {
	m.Selected = max(0, len(m.Rows)-1)
	m.keepVisible(height)
}

// ToggleSelected expands or collapses the selected directory. It returns true
// when the selected row was a directory.
func (m *Model) ToggleSelected() bool {
	if m.Selected < 0 || m.Selected >= len(m.Rows) || !m.Rows[m.Selected].Directory {
		return false
	}
	path := m.Rows[m.Selected].Path
	m.Expanded[path] = !m.Expanded[path]
	selectedPath := ""
	if index, ok := m.SelectedEntryIndex(); ok {
		selectedPath = string(m.Entries[index].Path)
	}
	m.Rows = buildRows(m.Entries, m.Visible, m.Expanded)
	if selectedPath != "" {
		for index, row := range m.Rows {
			if !row.Directory && row.Path == selectedPath {
				m.Selected = index
				break
			}
		}
	}
	m.Selected = min(m.Selected, max(0, len(m.Rows)-1))
	m.keepVisible(0)
	return true
}

// ExpandAll expands every directory currently represented by the filtered
// status entries.
func (m *Model) ExpandAll() {
	for _, row := range m.Rows {
		if row.Directory {
			m.Expanded[row.Path] = true
		}
	}
	m.Rows = buildRows(m.Entries, m.Visible, m.Expanded)
	m.Selected = min(m.Selected, max(0, len(m.Rows)-1))
}

// CollapseAll collapses every directory while retaining the root-level rows.
func (m *Model) CollapseAll() {
	for _, row := range m.Rows {
		if row.Directory {
			m.Expanded[row.Path] = false
		}
	}
	m.Rows = buildRows(m.Entries, m.Visible, m.Expanded)
	m.Selected = min(m.Selected, max(0, len(m.Rows)-1))
	m.Offset = 0
}

func (m *Model) keepVisible(height int) {
	if height <= 0 {
		return
	}
	if m.Selected < m.Offset {
		m.Offset = m.Selected
	}
	if m.Selected >= m.Offset+height {
		m.Offset = m.Selected - height + 1
	}
	m.Offset = max(0, min(m.Offset, max(0, len(m.Rows)-height)))
}

type node struct {
	path     string
	entry    int
	children map[string]*node
	order    []string
}

func buildRows(entries []repo.Entry, visible []int, expanded map[string]bool) []Row {
	root := &node{entry: -1, children: make(map[string]*node)}
	for _, entryIndex := range visible {
		if entryIndex < 0 || entryIndex >= len(entries) {
			continue
		}
		parts := strings.Split(string(entries[entryIndex].Path), "/")
		current := root
		path := ""
		for index, part := range parts {
			if path == "" {
				path = part
			} else {
				path += "/" + part
			}
			child, ok := current.children[part]
			if !ok {
				child = &node{path: path, entry: -1, children: make(map[string]*node)}
				current.children[part] = child
				current.order = append(current.order, part)
			}
			if index == len(parts)-1 {
				child.entry = entryIndex
			}
			current = child
		}
	}
	for path := range root.children {
		if _, ok := expanded[path]; !ok {
			expanded[path] = true
		}
	}
	rows := make([]Row, 0, len(visible))
	var visit func(*node, int)
	visit = func(current *node, depth int) {
		for _, name := range current.order {
			child := current.children[name]
			if len(child.children) == 0 {
				rows = append(rows, Row{Path: child.path, EntryIndex: child.entry, Depth: depth})
				continue
			}
			rows = append(rows, Row{Path: child.path, EntryIndex: -1, Depth: depth, Directory: true, Counts: aggregate(child, entries)})
			if expanded[child.path] {
				visit(child, depth+1)
			}
		}
	}
	visit(root, 0)
	return rows
}

func aggregate(current *node, entries []repo.Entry) (counts Counts) {
	if current.entry >= 0 && current.entry < len(entries) {
		entry := entries[current.entry]
		counts.Staged, counts.Unstaged = boolInt(entry.Staged), boolInt(entry.Unstaged)
		counts.Untracked, counts.Conflicted = boolInt(entry.Untracked), boolInt(entry.Conflicted)
	}
	for _, child := range current.children {
		childCounts := aggregate(child, entries)
		counts.Staged += childCounts.Staged
		counts.Unstaged += childCounts.Unstaged
		counts.Untracked += childCounts.Untracked
		counts.Conflicted += childCounts.Conflicted
	}
	return counts
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SortRows is exposed for deterministic tests and callers that need to
// compare independently-built presentation indexes.
func SortRows(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
}
