package registry

import (
	"sort"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/health"
	"github.com/sphireinc/git-watch/internal/repo"
)

// Row is a render-ready repository status row.
type Row struct {
	Repository        Repository
	Branch            string
	Dirty             int
	Staged            int
	Unstaged          int
	Untracked         int
	Conflicts         int
	Ahead             int
	Behind            int
	Stashes           int
	Worktrees         int
	Remotes           int
	Operation         string
	Attention         string
	Warnings          []string
	State             string
	Gitignore         GitignoreHealth
	Health            health.Summary
	RemoteFetchStatus string
	RemoteFetchAt     time.Time
	RemoteFetchError  string
	Activity          []int
}

// GitignoreHealth is a compact dashboard projection of one repository's
// authoritative .gitignore inspection.
type GitignoreHealth struct {
	Exists                                          bool
	Managed, Unmanaged, Partial, Attention, Updates int
}

// Rows converts refresh results into independently owned dashboard rows.
func Rows(results []StatusResult) []Row {
	rows := make([]Row, 0, len(results))
	for _, result := range results {
		snapshot := result.Snapshot
		healthSummary := result.Health
		if healthSummary.Source == "" {
			healthSummary = health.ComputeWithSubmoduleIssues(snapshot, result.Stashes, 0, health.CountSubmoduleIssues(snapshot), result.Warnings)
		}
		row := Row{Repository: result.Repository, Branch: snapshot.Branch.Name, Staged: snapshot.Counts.Staged, Unstaged: snapshot.Counts.Unstaged, Untracked: snapshot.Counts.Untracked, Conflicts: snapshot.Counts.Conflicted, Ahead: snapshot.Branch.Ahead, Behind: snapshot.Branch.Behind, Stashes: result.Stashes, Worktrees: result.Worktrees, Remotes: result.Remotes, Warnings: append([]string(nil), result.Warnings...), Gitignore: result.Gitignore, Health: healthSummary, RemoteFetchStatus: result.Repository.LastAutoFetchStatus, RemoteFetchAt: result.Repository.LastAutoFetch, RemoteFetchError: result.Repository.LastAutoFetchError}
		row.Dirty = row.Staged + row.Unstaged + row.Untracked
		if row.RemoteFetchStatus == "failed" {
			warning := "auto-fetch: " + row.RemoteFetchError
			if row.RemoteFetchError == "" {
				warning = "auto-fetch failed"
			}
			row.Warnings = append(row.Warnings, warning)
			if row.Attention == "" {
				row.Attention = warning
			}
			if row.Health.Severity != health.SeverityCritical {
				row.Health.Severity = health.SeverityWarning
			}
			row.Health.Attention = append(row.Health.Attention, warning)
		}
		if snapshot.Operation != nil {
			row.Operation = snapshot.Operation.Kind().String()
		}
		switch {
		case row.Conflicts > 0:
			row.Attention = "conflict"
		case row.Operation != "":
			row.Attention = row.Operation
		case result.OperationFailed:
			row.Attention = "operation failed"
		case row.Dirty > 0 || row.Ahead > 0 || row.Behind > 0:
			row.Attention = "dirty/diverged"
		case result.ProviderAttention:
			row.Attention = "provider stale"
		}
		if result.Skipped {
			row.State = "inactive"
		} else if result.Error != nil {
			row.State = "error"
		} else {
			row.State = "ready"
		}
		rows = append(rows, row)
	}
	return rows
}

// FilterRows returns rows whose path, group, or status matches query.
func FilterRows(rows []Row, query string) []Row {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]Row(nil), rows...)
	}
	filtered := make([]Row, 0, len(rows))
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.Repository.Name), query) || strings.Contains(strings.ToLower(row.Repository.Path), query) || strings.Contains(strings.ToLower(row.Branch), query) || strings.Contains(strings.ToLower(row.Operation), query) || strings.Contains(strings.ToLower(row.Attention), query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

// NeedsAttention reports whether a row has an actionable local or provider
// condition worth exposing as a command-palette jump target.
func (r Row) NeedsAttention() bool { return r.Attention != "" }

// SortKey identifies the field used to order repository rows.
type SortKey string

const (
	// SortName orders rows by repository name.
	SortName   SortKey = "name"
	SortDirty  SortKey = "dirty"
	SortAhead  SortKey = "ahead"
	SortBehind SortKey = "behind"
)

// SortRows returns a sorted copy of rows.
func SortRows(rows []Row, key SortKey, descending bool) []Row {
	sorted := append([]Row(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool {
		left, right := 0, 0
		switch key {
		case SortDirty:
			left, right = sorted[i].Dirty, sorted[j].Dirty
		case SortAhead:
			left, right = sorted[i].Ahead, sorted[j].Ahead
		case SortBehind:
			left, right = sorted[i].Behind, sorted[j].Behind
		default:
			leftName := strings.ToLower(sorted[i].Repository.Name)
			rightName := strings.ToLower(sorted[j].Repository.Name)
			if leftName == rightName {
				if descending {
					return sorted[i].Repository.Path > sorted[j].Repository.Path
				}
				return sorted[i].Repository.Path < sorted[j].Repository.Path
			}
			if descending {
				return leftName > rightName
			}
			return leftName < rightName
		}
		if left == right {
			if descending {
				return sorted[i].Repository.Path > sorted[j].Repository.Path
			}
			return sorted[i].Repository.Path < sorted[j].Repository.Path
		}
		if descending {
			return left > right
		}
		return left < right
	})
	return sorted
}

func SnapshotCounts(snapshot repo.Snapshot) (dirty, staged, unstaged, untracked, conflicts int) {
	return snapshot.Counts.Staged + snapshot.Counts.Unstaged + snapshot.Counts.Untracked, snapshot.Counts.Staged, snapshot.Counts.Unstaged, snapshot.Counts.Untracked, snapshot.Counts.Conflicted
}
