// Package pathhistoryview renders bounded history entries for one repository path.
package pathhistoryview

import (
	"fmt"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/pathhistory"
	"github.com/sphireinc/git-watch/internal/platform"
)

// Model stores only already-loaded path history and selection state.
type Model struct {
	Path     string
	Follow   bool
	Entries  []pathhistory.Entry
	Selected int
	HasMore  bool
}

func (m *Model) SetPage(path string, entries []pathhistory.Entry, hasMore bool) {
	m.Path = path
	m.Entries = append([]pathhistory.Entry(nil), entries...)
	m.Selected = 0
	m.HasMore = hasMore
}

func (m *Model) AppendPage(entries []pathhistory.Entry, hasMore bool) {
	m.Entries = append(m.Entries, entries...)
	m.HasMore = hasMore
}

func (m *Model) Move(delta int) {
	if len(m.Entries) == 0 {
		m.Selected = -1
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

func (m Model) SelectedEntry() (pathhistory.Entry, bool) {
	if m.Selected < 0 || m.Selected >= len(m.Entries) {
		return pathhistory.Entry{}, false
	}
	return m.Entries[m.Selected], true
}

func (m Model) View() string {
	path := clean(m.Path)
	follow := "exact path"
	if m.Follow {
		follow = "rename-following"
	}
	lines := []string{"Path history · " + path, "Mode: " + follow}
	for i, entry := range m.Entries {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		commit := entry.Commit.Short
		if commit == "" {
			commit = entry.Commit.SHA
		}
		when := ""
		if entry.Commit.Unix != 0 {
			when = time.Unix(entry.Commit.Unix, 0).Format(time.RFC3339)
		}
		line := fmt.Sprintf("%s%s · %s · %s", prefix, clean(commit), clean(entry.Kind), clean(entry.Commit.Subject))
		lines = append(lines, line)
		metadata := clean(entry.Commit.Author)
		if when != "" {
			metadata += " · " + when
		}
		if entry.OldPath != entry.NewPath && entry.NewPath != "" {
			metadata += " · " + clean(entry.OldPath) + " → " + clean(entry.NewPath)
		}
		lines = append(lines, "    "+metadata)
	}
	if len(m.Entries) == 0 {
		lines = append(lines, "  No history entries")
	}
	if m.HasMore {
		lines = append(lines, "", "[right bracket] load more history")
	}
	return strings.Join(lines, "\n")
}

func clean(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
	return platform.SafeText(value)
}
