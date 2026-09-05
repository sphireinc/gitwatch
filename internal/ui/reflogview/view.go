// Package reflogview renders bounded reflog recovery points.
package reflogview

import (
	"fmt"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/reflog"
)

// Model stores only selection and already-loaded page state.
type Model struct {
	Entries  []reflog.Entry
	Selected int
	Ref      string
	HasMore  bool
}

func New(ref string) Model { return Model{Ref: ref, Selected: -1} }

func (m *Model) SetPage(entries []reflog.Entry, hasMore bool) {
	m.Entries = append([]reflog.Entry(nil), entries...)
	m.Selected = 0
	m.HasMore = hasMore
}

func (m *Model) AppendPage(entries []reflog.Entry, hasMore bool) {
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

func (m Model) SelectedEntry() (reflog.Entry, bool) {
	if m.Selected < 0 || m.Selected >= len(m.Entries) {
		return reflog.Entry{}, false
	}
	return m.Entries[m.Selected], true
}

func (m Model) View() string {
	ref := m.Ref
	if ref == "" {
		ref = "HEAD"
	}
	lines := []string{"Reflog · " + platform.SafeText(ref), "Recovery points are local and expire according to Git configuration."}
	for index, entry := range m.Entries {
		prefix := "  "
		if index == m.Selected {
			prefix = "> "
		}
		when := time.Unix(entry.Timestamp, 0).Format(time.RFC3339)
		lines = append(lines, fmt.Sprintf("%s%s %s · %s · %s", prefix, safe(entry.SHA[:min(12, len(entry.SHA))]), when, safe(entry.Actor), safe(entry.Subject)))
	}
	if len(m.Entries) == 0 {
		lines = append(lines, "  No reflog entries")
	}
	if m.HasMore {
		lines = append(lines, "", "[right bracket] load more recovery points")
	}
	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func safe(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
	return platform.SafeText(value)
}
