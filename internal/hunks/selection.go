package hunks

import (
	"fmt"
	"github.com/sphireinc/git-watch/internal/patch"
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

func (s *Selection) SelectAll(files []patch.File) {
	s.Selected = make(map[ID]bool)
	for fileIndex, file := range files {
		for hunkIndex, hunk := range file.Hunks {
			s.SelectHunk(fileIndex, hunkIndex, hunk)
		}
	}
}

func (s *Selection) Invert(files []patch.File) {
	for fileIndex, file := range files {
		for hunkIndex, hunk := range file.Hunks {
			for lineIndex, line := range hunk.Lines {
				if line.Kind != patch.Added && line.Kind != patch.Removed {
					continue
				}
				id := ID{File: fileIndex, Hunk: hunkIndex, Line: lineIndex}
				if s.Selected[id] {
					delete(s.Selected, id)
				} else {
					s.Selected[id] = true
				}
			}
		}
	}
}

func (s *Selection) SelectRange(target ID, files []patch.File) {
	if s.Anchor == nil || !s.Valid(files) || !validID(target, files) || s.Anchor.File != target.File || s.Anchor.Hunk != target.Hunk {
		s.Toggle(target)
		return
	}
	start, end := s.Anchor.Line, target.Line
	if start > end {
		start, end = end, start
	}
	thunk := files[target.File].Hunks[target.Hunk]
	for line := start; line <= end; line++ {
		if kind := thunk.Lines[line].Kind; kind == patch.Added || kind == patch.Removed {
			s.Selected[ID{File: target.File, Hunk: target.Hunk, Line: line}] = true
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
		if !validID(id, files) {
			delete(s.Selected, id)
		}
	}
	if s.Anchor != nil && !s.Valid(files) {
		s.Anchor = nil
	}
}

func validID(id ID, files []patch.File) bool {
	return id.File >= 0 && id.File < len(files) && id.Hunk >= 0 && id.Hunk < len(files[id.File].Hunks) && id.Line >= 0 && id.Line < len(files[id.File].Hunks[id.Hunk].Lines)
}
func Identity(file, hunk, line int) string { return fmt.Sprintf("%d:%d:%d", file, hunk, line) }
