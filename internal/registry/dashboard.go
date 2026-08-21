package registry

import (
	"sort"
	"strings"

	"github.com/jusanchez/gitwatch/internal/repo"
)

type Row struct {
	Repository Repository
	Branch     string
	Dirty      int
	Staged     int
	Unstaged   int
	Untracked  int
	Conflicts  int
	Ahead      int
	Behind     int
	Stashes    int
	Remotes    int
	Warnings   []string
	State      string
}

func Rows(results []StatusResult) []Row {
	rows := make([]Row, 0, len(results))
	for _, result := range results {
		snapshot := result.Snapshot
		row := Row{Repository: result.Repository, Branch: snapshot.Branch.Name, Staged: snapshot.Counts.Staged, Unstaged: snapshot.Counts.Unstaged, Untracked: snapshot.Counts.Untracked, Conflicts: snapshot.Counts.Conflicted, Ahead: snapshot.Branch.Ahead, Behind: snapshot.Branch.Behind, Stashes: result.Stashes, Remotes: result.Remotes, Warnings: append([]string(nil), result.Warnings...)}
		row.Dirty = row.Staged + row.Unstaged + row.Untracked
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

func FilterRows(rows []Row, query string) []Row {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return append([]Row(nil), rows...)
	}
	filtered := make([]Row, 0, len(rows))
	for _, row := range rows {
		if strings.Contains(strings.ToLower(row.Repository.Name), query) || strings.Contains(strings.ToLower(row.Repository.Path), query) || strings.Contains(strings.ToLower(row.Branch), query) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

type SortKey string

const (
	SortName   SortKey = "name"
	SortDirty  SortKey = "dirty"
	SortAhead  SortKey = "ahead"
	SortBehind SortKey = "behind"
)

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
