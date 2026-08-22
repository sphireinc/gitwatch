package stashview

import (
	"fmt"
	"github.com/sphireinc/git-watch/internal/stash"
	"strings"
	"time"
)

// Model stores the selection state for the stash view.
type Model struct {
	Entries  []stash.Entry
	Selected int
	Filter   string
}

func New(e []stash.Entry) Model { return Model{Entries: e} }

func (m *Model) SetEntries(entries []stash.Entry) {
	selectedRef := ""
	if m.Selected >= 0 && m.Selected < len(m.Entries) {
		selectedRef = m.Entries[m.Selected].Ref
	}
	m.Entries = append([]stash.Entry(nil), entries...)
	m.Selected = 0
	for i, entry := range m.Entries {
		if entry.Ref == selectedRef {
			m.Selected = i
			break
		}
	}
}
func (m *Model) Move(d int) {
	m.Selected += d
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Entries) {
		m.Selected = len(m.Entries) - 1
	}
}
func (m Model) View() string {
	lines := []string{"Stashes"}
	for i, e := range m.Entries {
		if m.Filter != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(m.Filter)) {
			continue
		}
		p := "  "
		if i == m.Selected {
			p = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s · %s", p, e.Ref, e.Message))
		metadata := ""
		if e.Branch != "" {
			metadata = "branch: " + e.Branch
		}
		if e.OID != "" {
			if metadata != "" {
				metadata += " · "
			}
			metadata += "oid: " + e.OID
		}
		if e.Unix != 0 {
			if metadata != "" {
				metadata += " · "
			}
			metadata += "time: " + time.Unix(e.Unix, 0).Format(time.RFC3339)
		}
		if metadata != "" {
			lines = append(lines, "    "+metadata)
		}
	}
	if len(lines) == 1 {
		lines = append(lines, "  No stashes")
	}
	return strings.Join(lines, "\n")
}
