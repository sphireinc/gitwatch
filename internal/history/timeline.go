package history

import "strings"

type TimelineFilter struct {
	Repository string
	Kind       string
	Outcome    string
}

// OperationClass distinguishes read/network/provider work from operations
// that can rewrite local history. It is presentation metadata only; safety
// policy remains owned by each operation boundary.
func OperationClass(kind string) string {
	lower := strings.ToLower(strings.TrimSpace(kind))
	for _, background := range []string{"fetch", "auto-fetch", "provider", "github", "check", "issue", "release", "remote", "refresh", "load", "list", "blame", "history", "custom command"} {
		if strings.Contains(lower, background) {
			return "background"
		}
	}
	return "history mutation"
}

// ParseTimelineFilter accepts repository text plus optional field selectors:
// repo:<text>, type:<text>, and outcome:<text>. Unknown tokens remain part of
// the repository query so a plain search remains backwards compatible.
func ParseTimelineFilter(input string) TimelineFilter {
	filter := TimelineFilter{}
	var free []string
	for _, token := range strings.Fields(input) {
		key, value, ok := strings.Cut(token, ":")
		if !ok || strings.TrimSpace(value) == "" {
			free = append(free, token)
			continue
		}
		switch strings.ToLower(key) {
		case "repo", "repository":
			filter.Repository = value
		case "type", "kind":
			filter.Kind = value
		case "outcome", "result", "status":
			filter.Outcome = value
		default:
			free = append(free, token)
		}
	}
	if len(free) > 0 {
		filter.Repository = strings.TrimSpace(strings.Join(append(free, filter.Repository), " "))
	}
	return filter
}

// FilterEvents returns an independently owned bounded timeline projection.
// Empty fields are wildcards; matching is case-insensitive and repository
// scope is taken from the semantic operation record when present.
func FilterEvents(events []Event, filter TimelineFilter) []Event {
	repository := strings.ToLower(strings.TrimSpace(filter.Repository))
	kind := strings.ToLower(strings.TrimSpace(filter.Kind))
	outcome := strings.ToLower(strings.TrimSpace(filter.Outcome))
	filtered := make([]Event, 0, len(events))
	for _, event := range events {
		operationKind, operationOutcome, operationRepository := "", "", ""
		if event.Operation != nil {
			operationKind, operationOutcome, operationRepository = event.Operation.Kind, event.Operation.Outcome, event.Operation.Repository
		}
		if repository != "" && !strings.Contains(strings.ToLower(operationRepository), repository) {
			continue
		}
		if kind != "" && !strings.Contains(strings.ToLower(string(event.Kind)+" "+operationKind), kind) {
			continue
		}
		if outcome != "" && !strings.Contains(strings.ToLower(operationOutcome+" "+string(event.Kind)), outcome) {
			continue
		}
		filtered = append(filtered, cloneEvent(event))
	}
	return filtered
}
