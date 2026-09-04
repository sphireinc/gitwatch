package manage

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
)

var ErrNoAdoptableSegment = errors.New("template is not one exact contiguous unmanaged segment")

// PlanAdoptTemplate wraps an existing, contiguous template segment without
// changing any byte in that segment. Scattered or reordered rules are refused.
func PlanAdoptTemplate(snapshot domain.DocumentSnapshot, cat *catalog.Catalog, id domain.TemplateID) (domain.MutationPlan, error) {
	if cat == nil {
		return domain.MutationPlan{}, domain.ErrCatalogUnavailable
	}
	template, ok := cat.Get(id)
	if !ok {
		return domain.MutationPlan{}, fmt.Errorf("%w: %s", domain.ErrUnknownTemplate, id)
	}
	doc, err := document.Parse(snapshot.Bytes)
	if err != nil {
		return domain.MutationPlan{}, err
	}
	for _, result := range match.Match(doc, cat) {
		if result.TemplateID == id && result.Kind == domain.ManagedExact {
			return domain.MutationPlan{}, ErrAlreadyInstalled
		}
	}
	start, end, ok := contiguousSegment(doc, template.Content)
	if !ok {
		return domain.MutationPlan{}, ErrNoAdoptableSegment
	}
	newline := []byte("\n")
	if snapshot.Newline == domain.NewlineCRLF {
		newline = []byte("\r\n")
	}
	wrapped, err := wrapExact(id, cat.Version(), template.ContentSHA256, snapshot.Bytes[start:end], newline)
	if err != nil {
		return domain.MutationPlan{}, err
	}
	result := append(append(append([]byte(nil), snapshot.Bytes[:start]...), wrapped...), snapshot.Bytes[end:]...)
	return domain.NewMutationPlan(snapshot, domain.MutationUpdate, []domain.TemplateID{id}, []domain.Edit{{Start: start, End: end, Replacement: wrapped, TemplateID: id}}, result, nil)
}

func contiguousSegment(doc document.Document, content []byte) (int, int, bool) {
	pattern := splitLogical(content)
	if len(pattern) == 0 {
		return 0, 0, false
	}
	for i := 0; i+len(pattern) <= len(doc.Lines); i++ {
		matched := true
		for j, expected := range pattern {
			if !bytes.Equal(doc.Lines[i+j].Text, expected) {
				matched = false
				break
			}
		}
		if matched {
			return doc.Lines[i].Start, doc.Lines[i+len(pattern)-1].End, true
		}
	}
	return 0, 0, false
}

func splitLogical(content []byte) [][]byte {
	lines := bytes.Split(content, []byte("\n"))
	for len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	for i := range lines {
		lines[i] = bytes.TrimSuffix(lines[i], []byte("\r"))
	}
	return lines
}

func wrapExact(id domain.TemplateID, commit, contentHash string, body, newline []byte) ([]byte, error) {
	parsed, err := domain.ParseTemplateID(id.String())
	if err != nil {
		return nil, err
	}
	if commit == "" || contentHash == "" {
		return nil, managed.ErrMalformed
	}
	begin := []byte(fmt.Sprintf("# >>> gitwatch:gitignore begin format=%d id=%s source=github/gitignore commit=%s hash=%s", managed.FormatVersion, parsed, commit, contentHash))
	end := []byte(fmt.Sprintf("# <<< gitwatch:gitignore end format=%d id=%s", managed.FormatVersion, parsed))
	out := append(append(begin, newline...), body...)
	if !bytes.HasSuffix(out, newline) {
		out = append(out, newline...)
	}
	out = append(out, end...)
	return append(out, newline...), nil
}
