// Package committree parses bounded, colorized Git graph lines into safe
// semantic segments for themed rendering.
package committree

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Role identifies the semantic field represented by a segment.
type Role uint8

const (
	RolePlain Role = iota
	RoleGraph
	RoleHash
	RoleDecoration
	RoleSubject
	RoleDate
	RoleAuthor
)

// Segment is safe display text with a semantic theme role.
type Segment struct {
	Text string
	Role Role
}

// Parse converts Git SGR output into safe semantic segments. Unsupported
// controls and malformed escape sequences are discarded.
func Parse(line string) []Segment {
	var out []Segment
	role := RolePlain
	seenHash := false
	var text strings.Builder
	flush := func() {
		if text.Len() > 0 {
			out = append(out, Segment{Text: text.String(), Role: role})
			text.Reset()
		}
	}
	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			if end, ok := sgrEnd(line, i); ok {
				flush()
				role = roleForSGR(line[i+2:end], role)
				i = end + 1
				continue
			}
			if end, ok := csiEnd(line, i); ok {
				i = end
				continue
			}
			if end, ok := controlEnd(line, i); ok {
				i = end
				continue
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 1 {
			text.WriteRune('�')
			i++
			continue
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			if r == '\t' {
				text.WriteRune(r)
			}
			i += size
			continue
		}
		if !seenHash && role == RolePlain {
			role = RoleGraph
		}
		text.WriteRune(r)
		if role == RoleHash {
			seenHash = true
		}
		i += size
	}
	flush()
	return out
}

func sgrEnd(value string, start int) (int, bool) {
	if start+2 >= len(value) || value[start+1] != '[' {
		return 0, false
	}
	for i := start + 2; i < len(value); i++ {
		if value[i] == 'm' {
			return i, true
		}
		if value[i] < '0' || value[i] > '9' && value[i] != ';' {
			return 0, false
		}
	}
	return 0, false
}

func controlEnd(value string, start int) (int, bool) {
	if start+1 >= len(value) || value[start+1] != ']' {
		return 0, false
	}
	for i := start + 2; i < len(value); i++ {
		if value[i] == '\a' {
			return i + 1, true
		}
		if value[i] == 0x1b && i+1 < len(value) && value[i+1] == '\\' {
			return i + 2, true
		}
	}
	return len(value), true
}

func csiEnd(value string, start int) (int, bool) {
	if start+1 >= len(value) || value[start+1] != '[' {
		return 0, false
	}
	for i := start + 2; i < len(value); i++ {
		if value[i] >= 0x40 && value[i] <= 0x7e {
			return i + 1, true
		}
	}
	return len(value), true
}

func roleForSGR(value string, previous Role) Role {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" || value == "39" {
		return RolePlain
	}
	bold := false
	for _, part := range strings.Split(value, ";") {
		code, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		switch code {
		case 1:
			bold = true
		case 31:
			return RoleHash
		case 32:
			return RoleDate
		case 34:
			if bold {
				return RoleAuthor
			}
			return RoleDecoration
		case 33, 35, 36, 37, 90, 91, 92, 93, 94, 95, 96, 97:
			return RoleDecoration
		}
	}
	return previous
}
