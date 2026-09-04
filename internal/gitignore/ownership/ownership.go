// Package ownership analyzes exact gitignore rule ownership for safe removal.
package ownership

import (
	"bytes"
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
)

type OccurrenceKind string

const (
	ManagedOccurrence   OccurrenceKind = "managed"
	UnmanagedOccurrence OccurrenceKind = "unmanaged"
)

type Rule struct {
	Text         string
	Line         int
	Kind         OccurrenceKind
	ManagedBy    domain.TemplateID
	ReferencedBy []domain.TemplateID
}

type Decision struct {
	Rule         Rule
	SafeToRemove bool
	Reason       string
}

type Overlap struct {
	Rule      string
	Templates []domain.TemplateID
}

type Index struct{ rules []Rule }

type lineOwner struct {
	ID    domain.TemplateID
	Valid bool
}

func Build(doc document.Document, templates []catalog.Template, installed []domain.TemplateID) Index {
	installedSet := map[domain.TemplateID]bool{}
	for _, id := range installed {
		installedSet[id] = true
	}
	refs := map[string][]domain.TemplateID{}
	for _, template := range templates {
		for rule := range significant(template.Content) {
			refs[rule] = append(refs[rule], template.ID)
		}
	}
	for rule := range refs {
		sort.Slice(refs[rule], func(i, j int) bool { return refs[rule][i] < refs[rule][j] })
	}
	owners := managedLineOwners(doc)
	var rules []Rule
	for i, line := range doc.Lines {
		if line.Kind != document.Rule {
			continue
		}
		owner, managed := owners[i]
		kind := UnmanagedOccurrence
		var managedBy domain.TemplateID
		if managed && owner.Valid {
			kind, managedBy = ManagedOccurrence, owner.ID
		}
		rules = append(rules, Rule{Text: string(line.Text), Line: i, Kind: kind, ManagedBy: managedBy, ReferencedBy: append([]domain.TemplateID(nil), refs[string(line.Text)]...)})
	}
	_ = installedSet
	return Index{rules: rules}
}

func (i Index) Rules() []Rule {
	out := append([]Rule(nil), i.rules...)
	for n := range out {
		out[n].ReferencedBy = append([]domain.TemplateID(nil), out[n].ReferencedBy...)
	}
	return out
}

func (i Index) Removal(selected []domain.TemplateID, installed []domain.TemplateID) []Decision {
	selectedSet := map[domain.TemplateID]bool{}
	for _, id := range selected {
		selectedSet[id] = true
	}
	installedSet := map[domain.TemplateID]bool{}
	for _, id := range installed {
		installedSet[id] = true
	}
	out := make([]Decision, 0, len(i.rules))
	for _, rule := range i.rules {
		decision := Decision{Rule: rule}
		if rule.Kind == ManagedOccurrence {
			decision.SafeToRemove = selectedSet[rule.ManagedBy]
			if decision.SafeToRemove {
				decision.Reason = "inside selected managed block"
			} else {
				decision.Reason = "managed by an unselected block"
			}
			out = append(out, decision)
			continue
		}
		selectedRef, otherRef := false, false
		for _, id := range rule.ReferencedBy {
			if selectedSet[id] {
				selectedRef = true
			}
			if installedSet[id] && !selectedSet[id] {
				otherRef = true
			}
		}
		duplicates := 0
		for _, other := range i.rules {
			if other.Kind == UnmanagedOccurrence && other.Text == rule.Text {
				duplicates++
			}
		}
		switch {
		case !selectedRef:
			decision.Reason = "selected templates do not account for this rule"
		case otherRef:
			decision.Reason = "shared with an unselected installed template"
		case duplicates > 1:
			decision.Reason = "duplicate handwritten occurrences are ambiguous"
		default:
			decision.SafeToRemove = true
			decision.Reason = "unique exact unmanaged occurrence accounted for by selected templates"
		}
		out = append(out, decision)
	}
	return out
}

func (i Index) Overlaps() []Overlap {
	var out []Overlap
	for _, rule := range i.rules {
		if len(rule.ReferencedBy) < 2 {
			continue
		}
		ids := append([]domain.TemplateID(nil), rule.ReferencedBy...)
		out = append(out, Overlap{Rule: rule.Text, Templates: ids})
	}
	return out
}

func significant(content []byte) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range bytes.Split(content, []byte("\n")) {
		line = bytes.TrimSuffix(line, []byte("\r"))
		text := strings.TrimSpace(string(line))
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		out[string(line)] = struct{}{}
	}
	return out
}

func managedLineOwners(doc document.Document) map[int]lineOwner {
	out := map[int]lineOwner{}
	for i, line := range doc.Lines {
		value := strings.TrimSpace(string(line.Text))
		if !strings.HasPrefix(value, "# >>> gitwatch:gitignore begin ") {
			continue
		}
		for j := i + 1; j < len(doc.Lines); j++ {
			if !strings.HasPrefix(strings.TrimSpace(string(doc.Lines[j].Text)), "# <<< gitwatch:gitignore end ") {
				continue
			}
			block, err := managed.ParseManagedBlock(doc.Bytes[doc.Lines[i].Start:doc.Lines[j].End])
			if err == nil {
				for n := i + 1; n < j; n++ {
					if doc.Lines[n].Kind == document.Rule {
						out[n] = lineOwner{ID: block.TemplateID, Valid: true}
					}
				}
			}
			break
		}
	}
	return out
}
