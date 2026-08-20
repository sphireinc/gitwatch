package patch

import (
	"fmt"
	"strconv"
	"strings"
)

type LineKind byte

const (
	Context LineKind = ' '
	Added   LineKind = '+'
	Removed LineKind = '-'
	Meta    LineKind = 'm'
)

type Line struct {
	Kind             LineKind
	Text             string
	OldLine, NewLine int
	NoNewline        bool
}
type Hunk struct {
	OldStart, OldCount, NewStart, NewCount int
	Header                                 string
	Lines                                  []Line
}
type File struct {
	OldPath, NewPath                       string
	Binary                                 bool
	RenameFrom, RenameTo, CopyFrom, CopyTo string
	Hunks                                  []Hunk
	Raw                                    string
}
type ParseError struct{ Line, Reason string }

func (e *ParseError) Error() string { return fmt.Sprintf("patch: %s: %q", e.Reason, e.Line) }
func Parse(input string) ([]File, error) {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	var files []File
	var current *File
	var hunk *Hunk
	oldLine, newLine := 0, 0
	flush := func() {
		if current != nil {
			if hunk != nil {
				current.Hunks = append(current.Hunks, *hunk)
				hunk = nil
			}
			files = append(files, *current)
			current = nil
		}
	}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			parts := strings.Fields(line)
			if len(parts) < 4 {
				return nil, &ParseError{line, "malformed diff header"}
			}
			current = &File{OldPath: strings.TrimPrefix(parts[2], "a/"), NewPath: strings.TrimPrefix(parts[3], "b/"), Raw: line + "\n"}
		case current == nil:
			continue
		case strings.HasPrefix(line, "Binary files ") || strings.HasPrefix(line, "GIT binary patch"):
			current.Binary = true
			current.Raw += line + "\n"
		case strings.HasPrefix(line, "rename from "):
			current.RenameFrom = strings.TrimPrefix(line, "rename from ")
			current.Raw += line + "\n"
		case strings.HasPrefix(line, "rename to "):
			current.RenameTo = strings.TrimPrefix(line, "rename to ")
			current.Raw += line + "\n"
		case strings.HasPrefix(line, "copy from "):
			current.CopyFrom = strings.TrimPrefix(line, "copy from ")
			current.Raw += line + "\n"
		case strings.HasPrefix(line, "copy to "):
			current.CopyTo = strings.TrimPrefix(line, "copy to ")
			current.Raw += line + "\n"
		case strings.HasPrefix(line, "@@ "):
			if hunk != nil {
				current.Hunks = append(current.Hunks, *hunk)
			}
			a, b, c, d, err := header(line)
			if err != nil {
				return nil, &ParseError{line, err.Error()}
			}
			hunk = &Hunk{OldStart: a, OldCount: b, NewStart: c, NewCount: d, Header: line}
			oldLine, newLine = a, c
			current.Raw += line + "\n"
		case hunk != nil && line == "\\ No newline at end of file":
			hunk.Lines = append(hunk.Lines, Line{Kind: Meta, Text: line, NoNewline: true})
			current.Raw += line + "\n"
		case hunk != nil && (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")):
			kind := LineKind(line[0])
			l := Line{Kind: kind, Text: line[1:], OldLine: oldLine, NewLine: newLine}
			if kind != Added {
				oldLine++
			}
			if kind != Removed {
				newLine++
			}
			hunk.Lines = append(hunk.Lines, l)
			current.Raw += line + "\n"
		default:
			current.Raw += line + "\n"
		}
	}
	flush()
	return files, nil
}
func header(line string) (int, int, int, int, error) {
	f := strings.Fields(line)
	if len(f) < 3 {
		return 0, 0, 0, 0, fmt.Errorf("malformed hunk header")
	}
	parse := func(s string) (int, int, error) {
		s = strings.TrimPrefix(s, "-")
		s = strings.TrimPrefix(s, "+")
		p := strings.Split(s, ",")
		a, e := strconv.Atoi(p[0])
		if e != nil {
			return 0, 0, e
		}
		b := 1
		if len(p) > 1 {
			b, e = strconv.Atoi(p[1])
		}
		return a, b, e
	}
	a, b, e := parse(f[1])
	if e != nil {
		return 0, 0, 0, 0, e
	}
	c, d, e := parse(f[2])
	return a, b, c, d, e
}
