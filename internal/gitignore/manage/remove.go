package manage

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
	"github.com/sphireinc/git-watch/internal/gitignore/ownership"
)

var ErrModifiedManagedBlock = errors.New("managed block was modified and requires elevated confirmation")

type span struct {
	start, end int
	id         domain.TemplateID
}

// PlanRemoveTemplates creates one ownership-aware removal preview. It never
// deletes the .gitignore file, even when the resulting content is empty.
func PlanRemoveTemplates(snapshot domain.DocumentSnapshot, cat *catalog.Catalog, ids []domain.TemplateID) (domain.MutationPlan, error) {
	if cat == nil {
		return domain.MutationPlan{}, domain.ErrCatalogUnavailable
	}
	doc, err := document.Parse(snapshot.Bytes)
	if err != nil {
		return domain.MutationPlan{}, err
	}
	selected := uniqueSorted(ids)
	if len(selected) == 0 {
		return domain.MutationPlan{}, errors.New("at least one template is required")
	}
	selectedSet := map[domain.TemplateID]bool{}
	for _, id := range selected {
		if _, ok := cat.Get(id); !ok {
			return domain.MutationPlan{}, fmt.Errorf("%w: %s", domain.ErrUnknownTemplate, id)
		}
		selectedSet[id] = true
	}
	spans, edited := managedSpans(doc, selectedSet, cat)
	if edited != "" {
		return domain.MutationPlan{}, fmt.Errorf("%w: %s", ErrModifiedManagedBlock, edited)
	}
	installed := []domain.TemplateID{}
	for _, result := range match.Match(doc, cat) {
		if result.Kind.Full() {
			installed = append(installed, result.TemplateID)
		}
	}
	index := ownership.Build(doc, cat.List(), installed)
	decisions := index.Removal(selected, installed)
	for _, decision := range decisions {
		if decision.Rule.Kind == ownership.UnmanagedOccurrence && decision.SafeToRemove {
			spans = append(spans, span{start: doc.Lines[decision.Rule.Line].Start, end: doc.Lines[decision.Rule.Line].End})
		}
	}
	warnings := []string{}
	for _, decision := range decisions {
		if decision.Rule.Kind == ownership.UnmanagedOccurrence && !decision.SafeToRemove {
			if decision.Rule.ReferencedByAny(selectedSet) {
				warnings = append(warnings, fmt.Sprintf("line %d retained: %s", decision.Rule.Line+1, decision.Reason))
			}
		}
	}
	if len(spans) == 0 {
		return domain.MutationPlan{}, fmt.Errorf("%w: no safely removable content", domain.ErrAmbiguousUnmanagedRemoval)
	}
	spans = mergeSpans(spans)
	result := removeSpans(snapshot.Bytes, spans)
	edits := make([]domain.Edit, len(spans))
	for i, item := range spans {
		edits[i] = domain.Edit{Start: item.start, End: item.end, TemplateID: item.id}
	}
	return domain.NewMutationPlan(snapshot, domain.MutationRemove, selected, edits, result, warnings)
}

func uniqueSorted(ids []domain.TemplateID) []domain.TemplateID {
	seen := map[domain.TemplateID]bool{}
	out := []domain.TemplateID{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func managedSpans(doc document.Document, selected map[domain.TemplateID]bool, cat *catalog.Catalog) ([]span, string) {
	var out []span
	for i, line := range doc.Lines {
		if !strings.HasPrefix(strings.TrimSpace(string(line.Text)), "# >>> gitwatch:gitignore begin ") {
			continue
		}
		for j := i + 1; j < len(doc.Lines); j++ {
			if !strings.HasPrefix(strings.TrimSpace(string(doc.Lines[j].Text)), "# <<< gitwatch:gitignore end ") {
				continue
			}
			block, err := managed.ParseManagedBlock(doc.Bytes[doc.Lines[i].Start:doc.Lines[j].End])
			if err == nil && selected[block.TemplateID] {
				template, ok := cat.Get(block.TemplateID)
				if !ok || block.ContentSHA256 != template.ContentSHA256 {
					return nil, block.TemplateID.String()
				}
				out = append(out, span{start: doc.Lines[i].Start, end: doc.Lines[j].End, id: block.TemplateID})
			}
			break
		}
	}
	return out, ""
}

func mergeSpans(spans []span) []span {
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	out := []span{spans[0]}
	for _, item := range spans[1:] {
		last := &out[len(out)-1]
		if item.start <= last.end {
			if item.end > last.end {
				last.end = item.end
			}
			continue
		}
		out = append(out, item)
	}
	return out
}

func removeSpans(input []byte, spans []span) []byte {
	out := make([]byte, 0, len(input))
	cursor := 0
	for _, item := range spans {
		out = append(out, input[cursor:item.start]...)
		cursor = item.end
	}
	return append(out, input[cursor:]...)
}
