package hunkview

import (
	"fmt"
	"strings"

	"github.com/jusanchez/gitwatch/internal/hunks"
	"github.com/jusanchez/gitwatch/internal/patch"
)

type Model struct {
	Files            []patch.File
	File, Hunk, Line int
	Height, Offset   int
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
	if m.Hunk < 0 || m.Hunk >= len(p.Hunks) {
		m.Hunk, m.Line = 0, 0
	}
	m.Line += delta
	if m.Line < 0 {
		m.Line = 0
	}
	if m.Line >= len(p.Hunks[m.Hunk].Lines) {
		m.Line = len(p.Hunks[m.Hunk].Lines) - 1
	}
	m.ensureVisible()
}
func (m *Model) MoveHunk(delta int) {
	if len(m.Files) == 0 || delta == 0 {
		return
	}
	file, hunk := m.File, m.Hunk
	steps := len(m.Files) * 2
	for i := 0; i < steps; i++ {
		hunk += delta
		for hunk < 0 {
			file--
			if file < 0 {
				file = len(m.Files) - 1
			}
			hunk = len(m.Files[file].Hunks) - 1
		}
		for file < len(m.Files) && hunk >= len(m.Files[file].Hunks) {
			file++
			if file >= len(m.Files) {
				file = 0
			}
			hunk = 0
		}
		if file < len(m.Files) && len(m.Files[file].Hunks) > 0 {
			m.File, m.Hunk, m.Line, m.Offset = file, hunk, 0, 0
			return
		}
	}
}
func (m *Model) MoveFile(delta int) {
	if len(m.Files) == 0 || delta == 0 {
		return
	}
	file := m.File
	for i := 0; i < len(m.Files); i++ {
		file = (file + delta + len(m.Files)) % len(m.Files)
		if len(m.Files[file].Hunks) > 0 {
			m.File, m.Hunk, m.Line, m.Offset = file, 0, 0, 0
			return
		}
	}
}
func (m *Model) SelectLine(line int) bool {
	if len(m.Files) == 0 || m.File < 0 || m.File >= len(m.Files) || m.Hunk < 0 || m.Hunk >= len(m.Files[m.File].Hunks) {
		return false
	}
	if line < 0 || line >= len(m.Files[m.File].Hunks[m.Hunk].Lines) {
		return false
	}
	m.Line = line
	m.ensureVisible()
	if kind := m.Files[m.File].Hunks[m.Hunk].Lines[line].Kind; kind != patch.Added && kind != patch.Removed {
		return false
	}
	m.Selection.Toggle(hunks.ID{File: m.File, Hunk: m.Hunk, Line: line})
	return true
}
func (m *Model) Toggle() { m.SelectLine(m.Line) }

func (m *Model) SetHeight(height int) {
	if height < 0 {
		height = 0
	}
	m.Height = height
	m.ensureVisible()
}

func (m Model) LineAt(row int) int { return m.Offset + row }

func (m *Model) ensureVisible() {
	if m.Height <= 0 {
		m.Offset = 0
		return
	}
	if m.Line < m.Offset {
		m.Offset = m.Line
	}
	if m.Line >= m.Offset+m.Height {
		m.Offset = m.Line - m.Height + 1
	}
	if m.Offset < 0 {
		m.Offset = 0
	}
}
func (m Model) View() string {
	if len(m.Files) == 0 {
		return "No patch"
	}
	f := m.Files[m.File]
	if len(f.Hunks) == 0 || m.Hunk < 0 || m.Hunk >= len(f.Hunks) {
		return "No selectable hunks"
	}
	lines := []string{fmt.Sprintf("%s · hunk %d/%d · selected %d", f.NewPath, m.Hunk+1, len(f.Hunks), m.Selection.Count())}
	start, end := 0, len(f.Hunks[m.Hunk].Lines)
	if m.Height > 0 {
		start = m.Offset
		if start > end {
			start = end
		}
		end = start + m.Height
		if end > len(f.Hunks[m.Hunk].Lines) {
			end = len(f.Hunks[m.Hunk].Lines)
		}
	}
	for i, l := range f.Hunks[m.Hunk].Lines[start:end] {
		i += start
		mark := " "
		if m.Selection.Selected[hunks.ID{File: m.File, Hunk: m.Hunk, Line: i}] {
			mark = "✓"
		}
		lines = append(lines, fmt.Sprintf("%s %c %s", mark, l.Kind, l.Text))
	}
	lines = append(lines, "Space select · j/k move · a select hunk · Esc back")
	return strings.Join(lines, "\n")
}
