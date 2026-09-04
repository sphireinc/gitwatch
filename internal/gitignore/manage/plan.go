package manage

import (
	"bytes"
	"fmt"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

type Preview struct {
	Repository domain.RepositoryID
	Path       string
	Kind       domain.MutationKind
	Selected   []domain.TemplateID
	Warnings   []string
	Diff       string
}

func PreviewPlan(plan domain.MutationPlan) Preview {
	return Preview{Repository: plan.Repository, Path: plan.Path, Kind: plan.Kind, Selected: append([]domain.TemplateID(nil), plan.Selected...), Warnings: append([]string(nil), plan.Warnings...), Diff: unifiedDiff(plan.BeforeBytes, plan.ResultBytes)}
}

func unifiedDiff(before, after []byte) string {
	if bytes.Equal(before, after) {
		return ""
	}
	first := 0
	limit := len(before)
	if len(after) < limit {
		limit = len(after)
	}
	for first < limit && before[first] == after[first] {
		first++
	}
	beforeEnd, afterEnd := len(before), len(after)
	for beforeEnd > first && afterEnd > first && before[beforeEnd-1] == after[afterEnd-1] {
		beforeEnd--
		afterEnd--
	}
	return fmt.Sprintf("--- .gitignore (before)\n+++ .gitignore (after)\n@@ byte %d,%d -> %d,%d @@\n-%s+%s", first, beforeEnd-first, first, afterEnd-first, before[first:beforeEnd], after[first:afterEnd])
}
