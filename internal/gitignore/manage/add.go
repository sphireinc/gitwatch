// Package manage contains previewable, race-protected gitignore mutations.
package manage

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
)

var (
	ErrAlreadyInstalled = errors.New("gitignore template is already managed")
	ErrNoTemplatesToAdd = errors.New("no selected gitignore templates require adding")
)

// PlanAddTemplates creates a complete preview without writing. IDs are sorted
// to make multi-select output deterministic and existing bytes are retained
// verbatim before the single append edit.
func PlanAddTemplates(snapshot domain.DocumentSnapshot, cat *catalog.Catalog, ids []domain.TemplateID) (domain.MutationPlan, error) {
	if cat == nil {
		return domain.MutationPlan{}, domain.ErrCatalogUnavailable
	}
	doc, err := document.Parse(snapshot.Bytes)
	if err != nil {
		return domain.MutationPlan{}, err
	}
	results := match.Match(doc, cat)
	states := map[domain.TemplateID]match.Result{}
	for _, result := range results {
		states[result.TemplateID] = result
	}
	selected := append([]domain.TemplateID(nil), ids...)
	sort.Slice(selected, func(i, j int) bool { return selected[i] < selected[j] })
	var additions []domain.TemplateID
	warnings := []string{}
	seen := map[domain.TemplateID]bool{}
	for _, id := range selected {
		if seen[id] {
			continue
		}
		seen[id] = true
		template, ok := cat.Get(id)
		if !ok {
			return domain.MutationPlan{}, fmt.Errorf("%w: %s", domain.ErrUnknownTemplate, id)
		}
		state := states[id]
		switch state.Kind {
		case domain.ManagedExact:
			return domain.MutationPlan{}, fmt.Errorf("%w: %s", ErrAlreadyInstalled, id)
		case domain.ManagedEdited:
			return domain.MutationPlan{}, fmt.Errorf("%w: managed template %s has been edited", domain.ErrMalformedManagedBlock, id)
		case domain.UnmanagedFull:
			warnings = append(warnings, fmt.Sprintf("%s already present; no append needed", id))
		default:
			additions = append(additions, id)
			_ = template
		}
	}
	if len(additions) == 0 {
		return domain.MutationPlan{}, ErrNoTemplatesToAdd
	}
	newline := []byte("\n")
	if snapshot.Newline == domain.NewlineCRLF {
		newline = []byte("\r\n")
	}
	var appendBytes []byte
	for _, id := range additions {
		template, _ := cat.Get(id)
		block, err := managed.EncodeManagedBlock(id, "github/gitignore", cat.Version(), template.ContentSHA256, template.Content, newline)
		if err != nil {
			return domain.MutationPlan{}, err
		}
		if len(appendBytes) > 0 {
			appendBytes = append(appendBytes, newline...)
		}
		appendBytes = append(appendBytes, block...)
	}
	result := append([]byte(nil), snapshot.Bytes...)
	if len(result) > 0 {
		if result[len(result)-1] != '\n' && result[len(result)-1] != '\r' {
			result = append(result, newline...)
		}
		if !endsWithNewline(result, newline, 2) {
			result = append(result, newline...)
		}
	}
	result = append(result, appendBytes...)
	plan, err := domain.NewMutationPlan(snapshot, domain.MutationAppend, additions, []domain.Edit{{Start: len(snapshot.Bytes), End: len(snapshot.Bytes), Replacement: append([]byte(nil), result[len(snapshot.Bytes):]...)}}, result, warnings)
	if err != nil {
		return domain.MutationPlan{}, err
	}
	return plan, nil
}

func endsWithNewline(value, newline []byte, count int) bool {
	needed := len(newline) * count
	return len(value) >= needed && string(value[len(value)-needed:]) == string(append(append([]byte(nil), newline...), newline...))
}
