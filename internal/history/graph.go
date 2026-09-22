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

// GraphCursor carries the active lanes across a paginated history load. It is
// intentionally opaque to callers other than the graph builder so a page can
// continue the topology established by the preceding page.
type GraphCursor struct {
	lanes []string
}

// BuildGraphPage renders one page and returns the cursor needed by the next
// page. The cursor must be the one returned by the immediately preceding page
// for lane continuity to be preserved.
func BuildGraphPage(commits []Commit, cursor GraphCursor) ([]GraphRow, GraphCursor) {
	lanes := append([]string(nil), cursor.lanes...)
	rows := make([]GraphRow, 0, len(commits))
	for _, commit := range commits {
		lane := indexOf(lanes, commit.SHA)
		if lane < 0 {
			lane = len(lanes)
			lanes = append(lanes, commit.SHA)
		}
		row := GraphRow{Commit: commit, Lane: lane, Lanes: len(lanes), Parents: append([]string(nil), commit.Parents...)}
		for _, ref := range commit.Refs {
			ref = strings.TrimSpace(ref)
			switch {
			case strings.HasPrefix(ref, "HEAD -> "):
				row.Head = true
				row.Branches = append(row.Branches, strings.TrimPrefix(ref, "HEAD -> "))
			case ref == "HEAD":
				row.Head = true
			case strings.HasPrefix(ref, "tag: "):
				row.Tags = append(row.Tags, strings.TrimPrefix(ref, "tag: "))
			default:
				row.Branches = append(row.Branches, ref)
			}
		}
		rows = append(rows, row)
		lanes = advanceLanes(lanes, lane, commit.Parents)
	}
	return rows, GraphCursor{lanes: lanes}
}

// BuildGraph assigns stable lanes while walking commits newest-first. Existing
// parent lanes are reused; a merge keeps its first parent in place and places
// additional parents in newly allocated lanes.
func BuildGraph(commits []Commit) []GraphRow {
	rows, _ := BuildGraphPage(commits, GraphCursor{})
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
