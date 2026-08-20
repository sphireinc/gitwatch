package history

import "strings"

// GraphRow is the render-neutral representation of one commit in a lane graph.
// Connectors use ASCII/Unicode-free characters so the UI can select its theme.
type GraphRow struct {
	Commit   Commit
	Lane     int
	Lanes    int
	Parents  []string
	Branches []string
	Tags     []string
	Head     bool
}

// BuildGraph assigns stable lanes while walking commits newest-first. Existing
// parent lanes are reused; a merge keeps its first parent in place and places
// additional parents in newly allocated lanes.
func BuildGraph(commits []Commit) []GraphRow {
	lanes := make([]string, 0, len(commits))
	rows := make([]GraphRow, 0, len(commits))
	for _, commit := range commits {
		lane := indexOf(lanes, commit.SHA)
		if lane < 0 {
			lane = len(lanes)
			lanes = append(lanes, commit.SHA)
		}
		rows = append(rows, GraphRow{Commit: commit, Lane: lane, Lanes: len(lanes), Parents: append([]string(nil), commit.Parents...)})
		lanes = advanceLanes(lanes, lane, commit.Parents)
	}
	return rows
}

func advanceLanes(lanes []string, lane int, parents []string) []string {
	if lane < 0 || lane >= len(lanes) {
		return lanes
	}
	if len(parents) == 0 {
		return append(lanes[:lane], lanes[lane+1:]...)
	}
	lanes[lane] = parents[0]
	for _, parent := range parents[1:] {
		if indexOf(lanes, parent) < 0 {
			lanes = append(lanes, parent)
		}
	}
	return lanes
}

func indexOf(values []string, value string) int {
	for i, candidate := range values {
		if candidate == value {
			return i
		}
	}
	return -1
}

// Filter returns commits matching a case-insensitive subject, author, or SHA.
func Filter(commits []Commit, query string) []Commit {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]Commit(nil), commits...)
	}
	filtered := make([]Commit, 0, len(commits))
	for _, commit := range commits {
		if strings.Contains(strings.ToLower(commit.SHA), query) || strings.Contains(strings.ToLower(commit.Author), query) || strings.Contains(strings.ToLower(commit.Subject), query) {
			filtered = append(filtered, commit)
		}
	}
	return filtered
}
