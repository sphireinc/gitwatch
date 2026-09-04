// Package document parses .gitignore bytes without changing their spelling or
// line endings. It is intentionally separate from rendering and mutation.
package document

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

var ErrBinary = errors.New("gitignore contains NUL bytes and cannot be managed automatically")

type LineKind string

const (
	Blank         LineKind = "blank"
	Comment       LineKind = "comment"
	Rule          LineKind = "rule"
	ManagedMarker LineKind = "managed-marker"
)

type Line struct {
	Raw    []byte   `json:"raw"`
	Text   []byte   `json:"text"`
	Ending []byte   `json:"ending"`
	Start  int      `json:"start"`
	End    int      `json:"end"`
	Kind   LineKind `json:"kind"`
}

type ManagedBlock struct {
	TemplateID domain.TemplateID
	BeginLine  int
	EndLine    int
	Valid      bool
	Diagnostic string
}

type Document struct {
	Bytes           []byte
	Lines           []Line
	HasBOM          bool
	FinalNewline    bool
	DominantNewline domain.NewlineStyle
	Blocks          []ManagedBlock
}

func Parse(input []byte) (Document, error) {
	if bytes.IndexByte(input, 0) >= 0 {
		return Document{}, ErrBinary
	}
	doc := Document{Bytes: append([]byte(nil), input...), HasBOM: bytes.HasPrefix(input, []byte{0xef, 0xbb, 0xbf}), FinalNewline: len(input) > 0 && (input[len(input)-1] == '\n' || input[len(input)-1] == '\r'), DominantNewline: dominantNewline(input)}
	start := 0
	for start < len(input) {
		end := start
		for end < len(input) && input[end] != '\n' && input[end] != '\r' {
			end++
		}
		textEnd := end
		endingEnd := end
		if end < len(input) {
			if input[end] == '\r' && end+1 < len(input) && input[end+1] == '\n' {
				endingEnd += 2
			} else {
				endingEnd++
			}
		}
		text := input[start:textEnd]
		logical := text
		if start == 0 {
			logical = bytes.TrimPrefix(logical, []byte{0xef, 0xbb, 0xbf})
		}
		ending := input[textEnd:endingEnd]
		kind := classify(logical)
		if isMarkerCandidate(logical) {
			kind = ManagedMarker
		}
		doc.Lines = append(doc.Lines, Line{Raw: append([]byte(nil), input[start:endingEnd]...), Text: append([]byte(nil), logical...), Ending: append([]byte(nil), ending...), Start: start, End: endingEnd, Kind: kind})
		start = endingEnd
	}
	if len(input) == 0 {
		doc.Lines = []Line{}
	}
	doc.Blocks = scanBlocks(doc.Lines)
	return doc, nil
}

func (d Document) Render() []byte { return append([]byte(nil), d.Bytes...) }

func (d Document) LineAt(index int) (Line, bool) {
	if index < 0 || index >= len(d.Lines) {
		return Line{}, false
	}
	line := d.Lines[index]
	line.Raw = append([]byte(nil), line.Raw...)
	line.Text = append([]byte(nil), line.Text...)
	line.Ending = append([]byte(nil), line.Ending...)
	return line, true
}

func (d Document) Span(first, last int) ([]byte, error) {
	if first < 0 || last < first || last >= len(d.Lines) {
		return nil, fmt.Errorf("line span %d:%d out of range", first, last)
	}
	return append([]byte(nil), d.Bytes[d.Lines[first].Start:d.Lines[last].End]...), nil
}

func classify(text []byte) LineKind {
	if len(text) == 0 || (len(bytes.TrimSpace(text)) == 0) {
		return Blank
	}
	if text[0] == '#' {
		return Comment
	}
	return Rule
}

func isMarkerCandidate(text []byte) bool {
	trimmed := strings.TrimSpace(string(text))
	return strings.HasPrefix(trimmed, "# gitwatch:")
}

func scanBlocks(lines []Line) []ManagedBlock {
	var blocks []ManagedBlock
	open := -1
	invalid := false
	var id domain.TemplateID
	for i, line := range lines {
		if line.Kind != ManagedMarker {
			continue
		}
		value := strings.TrimSpace(string(line.Text))
		if strings.HasPrefix(value, "# gitwatch:begin ") {
			parsed, err := parseBegin(value)
			if open >= 0 {
				blocks = append(blocks, ManagedBlock{BeginLine: i, EndLine: i, Valid: false, Diagnostic: "nested managed begin marker"})
				invalid = true
				continue
			}
			open, id, invalid = i, parsed, err != nil
			continue
		}
		if strings.HasPrefix(value, "# gitwatch:end ") {
			endID, err := parseEnd(value)
			if open < 0 {
				blocks = append(blocks, ManagedBlock{BeginLine: i, EndLine: i, Valid: false, Diagnostic: "end marker without begin marker"})
				continue
			}
			valid := !invalid && err == nil && id != "" && endID == id
			diagnostic := ""
			if err != nil {
				diagnostic = err.Error()
			} else if endID != id {
				diagnostic = "end marker template does not match begin marker"
			}
			blocks = append(blocks, ManagedBlock{TemplateID: id, BeginLine: open, EndLine: i, Valid: valid, Diagnostic: diagnostic})
			open, id, invalid = -1, "", false
			continue
		}
		blocks = append(blocks, ManagedBlock{BeginLine: i, EndLine: i, Valid: false, Diagnostic: "malformed managed marker"})
	}
	if open >= 0 {
		blocks = append(blocks, ManagedBlock{TemplateID: id, BeginLine: open, EndLine: open, Valid: false, Diagnostic: "begin marker without end marker"})
	}
	return blocks
}

func parseBegin(value string) (domain.TemplateID, error) {
	parts := strings.Fields(strings.TrimPrefix(value, "# gitwatch:begin "))
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "template=") || parts[1] != "version=1" {
		return "", errors.New("malformed begin marker")
	}
	return domain.ParseTemplateID(strings.TrimPrefix(parts[0], "template="))
}

func parseEnd(value string) (domain.TemplateID, error) {
	parts := strings.Fields(strings.TrimPrefix(value, "# gitwatch:end "))
	if len(parts) != 1 || !strings.HasPrefix(parts[0], "template=") {
		return "", errors.New("malformed end marker")
	}
	return domain.ParseTemplateID(strings.TrimPrefix(parts[0], "template="))
}

func dominantNewline(input []byte) domain.NewlineStyle {
	lf, crlf := bytes.Count(input, []byte{'\n'}), bytes.Count(input, []byte{'\r', '\n'})
	if lf == 0 {
		return domain.NewlineNone
	}
	if crlf == lf {
		return domain.NewlineCRLF
	}
	if crlf == 0 {
		return domain.NewlineLF
	}
	if crlf*2 >= lf {
		return domain.NewlineCRLF
	}
	return domain.NewlineLF
}
