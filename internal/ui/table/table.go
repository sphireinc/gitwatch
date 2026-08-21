package table

import (
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/repo"
)

type SortMode uint8

const (
	SortPath SortMode = iota
	SortStatus
	SortStagedFirst
	SortChangedMost
)

type Model struct {
	Entries  []repo.Entry
	Visible  []int
	Selected int
	Filter   string
	Sort     SortMode
	Offset   int
}

func New(entries []repo.Entry) Model {
	m := Model{Entries: append([]repo.Entry(nil), entries...)}
	m.rebuild("")
	return m
}
func (m *Model) SetEntries(entries []repo.Entry) {
	selected := m.SelectedPath()
	m.Entries = append([]repo.Entry(nil), entries...)
	m.rebuild(selected)
}
func (m *Model) SetFilter(filter string) { m.Filter = filter; m.rebuild(m.SelectedPath()) }
func (m *Model) SetConflictFilter(enabled bool) {
	if enabled {
		m.Filter = "!conflict"
	} else {
		m.Filter = ""
	}
	m.rebuild(m.SelectedPath())
}
func (m *Model) CycleSort() { m.Sort = (m.Sort + 1) % 4; m.rebuild(m.SelectedPath()) }
func (m Model) SelectedPath() string {
	if m.Selected < 0 || m.Selected >= len(m.Visible) {
		return ""
	}
	return string(m.Entries[m.Visible[m.Selected]].Path)
}
func (m *Model) Move(delta, height int) {
	if len(m.Visible) == 0 {
		return
	}
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Visible) {
		m.Selected = len(m.Visible) - 1
	}
	if height > 0 {
		if m.Selected < m.Offset {
			m.Offset = m.Selected
		}
		if m.Selected >= m.Offset+height {
			m.Offset = m.Selected - height + 1
		}
	}
}
func (m Model) RowAt(y, top, height int) (repo.Entry, bool) {
	i := y - top + m.Offset
	if i < 0 || i >= height || i >= len(m.Visible) {
		return repo.Entry{}, false
	}
	return m.Entries[m.Visible[i]], true
}

func (m *Model) rebuild(previous string) {
	m.Visible = m.Visible[:0]
	for i, entry := range m.Entries {
		if m.Filter == "!conflict" {
			if entry.Conflicted {
				m.Visible = append(m.Visible, i)
			}
			continue
		}
		if fuzzy(string(entry.Path), m.Filter) {
			m.Visible = append(m.Visible, i)
		}
	}
	sort.SliceStable(m.Visible, func(a, b int) bool {
		x, y := m.Entries[m.Visible[a]], m.Entries[m.Visible[b]]
		switch m.Sort {
		case SortStatus:
			return repo.StatusLabel(x) < repo.StatusLabel(y)
		case SortStagedFirst:
			if x.Staged != y.Staged {
				return x.Staged
			}
		case SortChangedMost:
			return changed(x) > changed(y)
		}
		return string(x.Path) < string(y.Path)
	})
	m.Selected = 0
	for i, index := range m.Visible {
		if string(m.Entries[index].Path) == previous {
			m.Selected = i
			break
		}
	}
	if m.Selected >= len(m.Visible) {
		m.Selected = max(0, len(m.Visible)-1)
	}
	m.Offset = 0
}
func changed(e repo.Entry) int {
	n := 0
	if e.Staged {
		n++
	}
	if e.Unstaged {
		n++
	}
	if e.Deleted {
		n += 2
	}
	return n
}
func fuzzy(value, query string) bool {
	if query == "" {
		return true
	}
	value = strings.ToLower(value)
	query = strings.ToLower(query)
	at := 0
	for _, r := range query {
		i := strings.IndexRune(value[at:], r)
		if i < 0 {
			return false
		}
		at += i + 1
	}
	return true
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
