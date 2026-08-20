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
	var raw strings.Builder
	oldLine, newLine := 0, 0
	flush := func() {
		if current != nil {
			if hunk != nil {
				current.Hunks = append(current.Hunks, *hunk)
				hunk = nil
			}
			current.Raw = raw.String()
			files = append(files, *current)
			current = nil
			raw.Reset()
		}
	}
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			oldPath, newPath, err := diffHeaderPaths(line)
			if err != nil {
				return nil, &ParseError{line, err.Error()}
			}
			current = &File{OldPath: oldPath, NewPath: newPath}
			raw.WriteString(line)
			raw.WriteByte('\n')
		case current == nil:
			continue
		case strings.HasPrefix(line, "Binary files ") || strings.HasPrefix(line, "GIT binary patch"):
			current.Binary = true
			raw.WriteString(line)
			raw.WriteByte('\n')
		case strings.HasPrefix(line, "rename from "):
			current.RenameFrom = strings.TrimPrefix(line, "rename from ")
			raw.WriteString(line)
			raw.WriteByte('\n')
		case strings.HasPrefix(line, "rename to "):
			current.RenameTo = strings.TrimPrefix(line, "rename to ")
			raw.WriteString(line)
			raw.WriteByte('\n')
		case strings.HasPrefix(line, "copy from "):
			current.CopyFrom = strings.TrimPrefix(line, "copy from ")
			raw.WriteString(line)
			raw.WriteByte('\n')
		case strings.HasPrefix(line, "copy to "):
			current.CopyTo = strings.TrimPrefix(line, "copy to ")
			raw.WriteString(line)
			raw.WriteByte('\n')
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
			raw.WriteString(line)
			raw.WriteByte('\n')
		case hunk != nil && line == "\\ No newline at end of file":
			hunk.Lines = append(hunk.Lines, Line{Kind: Meta, Text: line, NoNewline: true})
			raw.WriteString(line)
			raw.WriteByte('\n')
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
			raw.WriteString(line)
			raw.WriteByte('\n')
		default:
			raw.WriteString(line)
			raw.WriteByte('\n')
		}
	}
	flush()
	return files, nil
}

func diffHeaderPaths(line string) (string, string, error) {
	rest := strings.TrimPrefix(line, "diff --git ")
	if rest == line {
		return "", "", fmt.Errorf("malformed diff header")
	}
	if strings.HasPrefix(rest, "a/") {
		if separator := strings.Index(rest, " b/"); separator >= 0 {
			old, newPath := rest[:separator], rest[separator+1:]
			return strings.TrimPrefix(old, "a/"), strings.TrimPrefix(newPath, "b/"), nil
		}
	}
	old, remainder, err := headerPathToken(rest)
	if err != nil {
		return "", "", err
	}
	remainder = strings.TrimLeft(remainder, " \t")
	newPath, remainder, err := headerPathToken(remainder)
	if err != nil || strings.TrimSpace(remainder) != "" {
		return "", "", fmt.Errorf("malformed diff header")
	}
	if !strings.HasPrefix(old, "a/") || !strings.HasPrefix(newPath, "b/") {
		return "", "", fmt.Errorf("diff header paths must use a/ and b/ prefixes")
	}
	return strings.TrimPrefix(old, "a/"), strings.TrimPrefix(newPath, "b/"), nil
}

func headerPathToken(input string) (string, string, error) {
	if input == "" {
		return "", "", fmt.Errorf("malformed diff header")
	}
	if input[0] == '"' {
		end := 1
		escaped := false
		for ; end < len(input); end++ {
			if input[end] == '"' && !escaped {
				break
			}
			if input[end] == '\\' && !escaped {
				escaped = true
			} else {
				escaped = false
			}
		}
		if end >= len(input) {
			return "", "", fmt.Errorf("malformed quoted path")
		}
		value, err := strconv.Unquote(input[:end+1])
		if err != nil {
			return "", "", fmt.Errorf("malformed quoted path")
		}
		return value, input[end+1:], nil
	}
	if strings.HasPrefix(input, "a/") {
		if separator := strings.Index(input, " b/"); separator >= 0 {
			return input[:separator], input[separator+1:], nil
		}
	}
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", "", fmt.Errorf("malformed diff header")
	}
	return fields[0], input[len(fields[0]):], nil
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
