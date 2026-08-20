package hunkview

import (
	"fmt"
	"github.com/jusanchez/gitwatch/internal/hunks"
	"github.com/jusanchez/gitwatch/internal/patch"
	"strings"
)

type Model struct {
	Files            []patch.File
	File, Hunk, Line int
	Selection        hunks.Selection
}

func New(files []patch.File) Model { return Model{Files: files, Selection: hunks.New()} }
func (m *Model) Move(delta int) {
	if len(m.Files) == 0 {
		return
	}
	p := m.Files[m.File]
	if len(p.Hunks) == 0 {
		return
	}
	m.Line += delta
	if m.Line < 0 {
		m.Line = 0
	}
	if m.Line >= len(p.Hunks[m.Hunk].Lines) {
		m.Line = len(p.Hunks[m.Hunk].Lines) - 1
	}
}
func (m *Model) Toggle() { m.Selection.Toggle(hunks.ID{File: m.File, Hunk: m.Hunk, Line: m.Line}) }
func (m Model) View() string {
	if len(m.Files) == 0 {
		return "No patch"
	}
	f := m.Files[m.File]
	if len(f.Hunks) == 0 || m.Hunk < 0 || m.Hunk >= len(f.Hunks) {
		return "No selectable hunks"
	}
	lines := []string{fmt.Sprintf("%s · hunk %d/%d · selected %d", f.NewPath, m.Hunk+1, len(f.Hunks), m.Selection.Count())}
	for i, l := range f.Hunks[m.Hunk].Lines {
		mark := " "
		if m.Selection.Selected[hunks.ID{File: m.File, Hunk: m.Hunk, Line: i}] {
			mark = "✓"
		}
		lines = append(lines, fmt.Sprintf("%s %c %s", mark, l.Kind, l.Text))
	}
	lines = append(lines, "Space select · j/k move · a select hunk · Esc back")
	return strings.Join(lines, "\n")
}
