package commands

import "strings"

type Action struct {
	ID       string
	Label    string
	Category string
	Shortcut string
	Enabled  bool
	Reason   string
	Run      func() error
}

type Match struct {
	Action
	Score int
}

func Search(actions []Action, query string) []Match {
	category, query := parseQuery(query)
	var matches []Match
	for _, action := range actions {
		if category != "" && strings.ToLower(action.Category) != category {
			continue
		}
		if query == "" {
			matches = append(matches, Match{Action: action, Score: 0})
			continue
		}
		score, ok := subsequenceScore(strings.ToLower(action.Label), query)
		if !ok {
			continue
		}
		matches = append(matches, Match{Action: action, Score: score})
	}
	for i := 1; i < len(matches); i++ {
		value := matches[i]
		j := i - 1
		for ; j >= 0 && (value.Score < matches[j].Score || value.Score == matches[j].Score && value.Label < matches[j].Label); j-- {
			matches[j+1] = matches[j]
		}
		matches[j+1] = value
	}
	return matches
}

func parseQuery(query string) (category, text string) {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	remaining := make([]string, 0, len(parts))
	for _, part := range parts {
		switch {
		case strings.HasPrefix(part, "repo:") || strings.HasPrefix(part, "repository:"):
			category = "repository"
			if suffix := strings.TrimPrefix(strings.TrimPrefix(part, "repo:"), "repository:"); suffix != "" {
				remaining = append(remaining, suffix)
			}
		case strings.HasPrefix(part, "category:"):
			category = strings.TrimPrefix(part, "category:")
		default:
			remaining = append(remaining, part)
		}
	}
	return category, strings.Join(remaining, " ")
}

func subsequenceScore(value, query string) (int, bool) {
	characters := []rune(value)
	position, score := 0, 0
	last := -1
	for _, needle := range query {
		found := -1
		for position < len(characters) {
			if characters[position] == needle {
				found = position
				position++
				break
			}
			position++
		}
		if found < 0 {
			return 0, false
		}
		if last >= 0 {
			score += found - last - 1
		}
		last = found
	}
	return score, true
}
