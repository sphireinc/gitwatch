package manage

import (
	"fmt"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

type MixedActionKind string

const (
	ActionAdd    MixedActionKind = "add"
	ActionRemove MixedActionKind = "remove"
	ActionUpdate MixedActionKind = "update"
	ActionAdopt  MixedActionKind = "adopt"
)

type MixedAction struct {
	ID   domain.TemplateID
	Kind MixedActionKind
}

type MixedSummary struct{ Add, Remove, Update, Adopt int }

func SummarizeMixed(actions []MixedAction) MixedSummary {
	var summary MixedSummary
	for _, action := range actions {
		switch action.Kind {
		case ActionAdd:
			summary.Add++
		case ActionRemove:
			summary.Remove++
		case ActionUpdate:
			summary.Update++
		case ActionAdopt:
			summary.Adopt++
		}
	}
	return summary
}

func (s MixedSummary) String() string {
	parts := []string{}
	if s.Add > 0 {
		parts = append(parts, fmt.Sprintf("%d add", s.Add))
	}
	if s.Remove > 0 {
		parts = append(parts, fmt.Sprintf("%d remove", s.Remove))
	}
	if s.Update > 0 {
		parts = append(parts, fmt.Sprintf("%d update", s.Update))
	}
	if s.Adopt > 0 {
		parts = append(parts, fmt.Sprintf("%d adopt", s.Adopt))
	}
	return joinSummary(parts)
}

func joinSummary(parts []string) string {
	out := ""
	for i, part := range parts {
		if i > 0 {
			out += ", "
		}
		out += part
	}
	return out
}
