package hunks

import (
	"fmt"
	"github.com/jusanchez/gitwatch/internal/patch"
)

type ID struct{ File, Hunk, Line int }
type Selection struct {
	Selected map[ID]bool
	Anchor   *ID
}

func New() Selection { return Selection{Selected: make(map[ID]bool)} }
func (s *Selection) Toggle(id ID) {
	if s.Selected[id] {
		delete(s.Selected, id)
	} else {
		s.Selected[id] = true
	}
	a := id
	s.Anchor = &a
}
func (s *Selection) SelectHunk(file, hunk int, p patch.Hunk) {
	for i, l := range p.Lines {
		if l.Kind == patch.Added || l.Kind == patch.Removed {
			s.Selected[ID{file, hunk, i}] = true
		}
	}
}
func (s *Selection) Clear()    { s.Selected = make(map[ID]bool); s.Anchor = nil }
func (s Selection) Count() int { return len(s.Selected) }
func (s Selection) Valid(files []patch.File) bool {
	for id := range s.Selected {
		if id.File < 0 || id.File >= len(files) || id.Hunk < 0 || id.Hunk >= len(files[id.File].Hunks) || id.Line < 0 || id.Line >= len(files[id.File].Hunks[id.Hunk].Lines) {
			return false
		}
	}
	return true
}
func (s *Selection) Invalidate(files []patch.File) {
	for id := range s.Selected {
		if id.File >= len(files) || id.Hunk >= len(files[id.File].Hunks) || id.Line >= len(files[id.File].Hunks[id.Hunk].Lines) {
			delete(s.Selected, id)
		}
	}
	if s.Anchor != nil && !s.Valid(files) {
		s.Anchor = nil
	}
}
func Identity(file, hunk, line int) string { return fmt.Sprintf("%d:%d:%d", file, hunk, line) }
