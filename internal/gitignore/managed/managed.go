// Package managed defines the human-readable, versioned gitwatch ownership
// block embedded in .gitignore documents.
package managed

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

const FormatVersion = 1

var (
	ErrMalformed     = errors.New("malformed gitwatch managed block")
	ErrUnknownFormat = errors.New("unknown gitwatch managed block format")
	ErrMismatchedID  = errors.New("managed block begin/end template IDs differ")
)

type Block struct {
	TemplateID    domain.TemplateID
	Source        string
	Commit        string
	ContentSHA256 string
	Body          []byte
	Format        int
}

func EncodeManagedBlock(id domain.TemplateID, source, commit, contentHash string, body []byte, newline []byte) ([]byte, error) {
	parsed, err := domain.ParseTemplateID(id.String())
	if err != nil {
		return nil, err
	}
	if source == "" || commit == "" || contentHash == "" {
		return nil, ErrMalformed
	}
	if len(newline) == 0 {
		newline = []byte("\n")
	}
	body = adaptNewlines(body, newline)
	begin := fmt.Sprintf("# >>> gitwatch:gitignore begin format=%d id=%s source=%s commit=%s hash=%s", FormatVersion, parsed, source, commit, contentHash)
	end := fmt.Sprintf("# <<< gitwatch:gitignore end format=%d id=%s", FormatVersion, parsed)
	result := append([]byte(begin), newline...)
	result = append(result, body...)
	if len(body) > 0 && !bytes.HasSuffix(body, newline) {
		result = append(result, newline...)
	}
	result = append(result, []byte(end)...)
	result = append(result, newline...)
	return result, nil
}

func ParseManagedBlock(input []byte) (Block, error) {
	lines := bytes.SplitAfter(input, []byte("\n"))
	if len(lines) < 2 {
		return Block{}, ErrMalformed
	}
	begin := strings.TrimSuffix(string(lines[0]), "\n")
	begin = strings.TrimSuffix(begin, "\r")
	endIndex := len(lines) - 1
	if len(lines[endIndex]) == 0 {
		endIndex--
	}
	end := strings.TrimSpace(string(lines[endIndex]))
	if !strings.HasPrefix(begin, "# >>> gitwatch:gitignore begin ") || !strings.HasPrefix(end, "# <<< gitwatch:gitignore end ") {
		return Block{}, ErrMalformed
	}
	meta, err := parseFields(strings.TrimPrefix(begin, "# >>> gitwatch:gitignore begin "))
	if err != nil {
		return Block{}, err
	}
	endMeta, err := parseFields(strings.TrimPrefix(end, "# <<< gitwatch:gitignore end "))
	if err != nil {
		return Block{}, err
	}
	if meta["format"] != fmt.Sprint(FormatVersion) || endMeta["format"] != fmt.Sprint(FormatVersion) {
		return Block{}, ErrUnknownFormat
	}
	id, err := domain.ParseTemplateID(meta["id"])
	if err != nil {
		return Block{}, ErrMalformed
	}
	if endMeta["id"] != id.String() {
		return Block{}, ErrMismatchedID
	}
	bodyStart := len(lines[0])
	bodyEnd := len(input) - len(lines[endIndex])
	if bodyEnd < bodyStart {
		return Block{}, ErrMalformed
	}
	return Block{TemplateID: id, Source: meta["source"], Commit: meta["commit"], ContentSHA256: meta["hash"], Body: append([]byte(nil), input[bodyStart:bodyEnd]...), Format: FormatVersion}, nil
}

func Validate(block Block) error {
	if block.Format != FormatVersion {
		return ErrUnknownFormat
	}
	if block.TemplateID == "" || block.Source == "" || block.Commit == "" || block.ContentSHA256 == "" {
		return ErrMalformed
	}
	return nil
}

func parseFields(value string) (map[string]string, error) {
	out := map[string]string{}
	for _, field := range strings.Fields(value) {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, ErrMalformed
		}
		out[parts[0]] = parts[1]
	}
	if out["format"] == "" || out["id"] == "" {
		return nil, ErrMalformed
	}
	return out, nil
}

func adaptNewlines(body, newline []byte) []byte {
	body = bytes.ReplaceAll(body, []byte("\r\n"), []byte("\n"))
	body = bytes.ReplaceAll(body, []byte("\r"), []byte("\n"))
	if bytes.Equal(newline, []byte("\n")) {
		return body
	}
	return bytes.ReplaceAll(body, []byte("\n"), newline)
}
