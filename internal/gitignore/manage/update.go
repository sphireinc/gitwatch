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
)

var ErrNoStaleManagedBlock = errors.New("no stale managed block selected for update")

// PlanUpdateTemplates replaces only stale managed blocks with the current
// bundled body and metadata. Handwritten bytes outside those blocks remain
// untouched.
func PlanUpdateTemplates(snapshot domain.DocumentSnapshot, cat *catalog.Catalog, ids []domain.TemplateID) (domain.MutationPlan, error) {
	if cat == nil {
		return domain.MutationPlan{}, domain.ErrCatalogUnavailable
	}
	doc, err := document.Parse(snapshot.Bytes)
	if err != nil {
		return domain.MutationPlan{}, err
	}
	selected := map[domain.TemplateID]bool{}
	for _, id := range uniqueSorted(ids) {
		if _, ok := cat.Get(id); !ok {
			return domain.MutationPlan{}, fmt.Errorf("%w: %s", domain.ErrUnknownTemplate, id)
		}
		selected[id] = true
	}
	newline := []byte("\n")
	if snapshot.Newline == domain.NewlineCRLF {
		newline = []byte("\r\n")
	}
	edits := []domain.Edit{}
	updated := []domain.TemplateID{}
	for i, line := range doc.Lines {
		if !strings.HasPrefix(strings.TrimSpace(string(line.Text)), "# >>> gitwatch:gitignore begin ") {
			continue
		}
		for j := i + 1; j < len(doc.Lines); j++ {
			if !strings.HasPrefix(strings.TrimSpace(string(doc.Lines[j].Text)), "# <<< gitwatch:gitignore end ") {
				continue
			}
			block, parseErr := managed.ParseManagedBlock(doc.Bytes[line.Start:doc.Lines[j].End])
			if parseErr == nil && selected[block.TemplateID] {
				template, _ := cat.Get(block.TemplateID)
				if block.ContentSHA256 != template.ContentSHA256 {
					replacement, encodeErr := managed.EncodeManagedBlock(template.ID, "github/gitignore", cat.Version(), template.ContentSHA256, template.Content, newline)
					if encodeErr != nil {
						return domain.MutationPlan{}, encodeErr
					}
					edits = append(edits, domain.Edit{Start: line.Start, End: doc.Lines[j].End, Replacement: replacement, TemplateID: block.TemplateID})
					updated = append(updated, block.TemplateID)
				}
			}
			break
		}
	}
	if len(updated) == 0 {
		return domain.MutationPlan{}, ErrNoStaleManagedBlock
	}
	result := append([]byte(nil), snapshot.Bytes...)
	sort.Slice(edits, func(i, j int) bool { return edits[i].Start > edits[j].Start })
	for _, edit := range edits {
		result = append(append(append([]byte(nil), result[:edit.Start]...), edit.Replacement...), result[edit.End:]...)
	}
	return domain.NewMutationPlan(snapshot, domain.MutationUpdate, updated, edits, result, nil)
}
