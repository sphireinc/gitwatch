package registry

import (
	"testing"

	"github.com/jusanchez/gitwatch/internal/repo"
)

func TestRowsFilterAndSort(t *testing.T) {
	rows := Rows([]StatusResult{{Repository: Repository{Name: "zeta", Path: "/z"}, Snapshot: repo.Snapshot{Branch: repo.Branch{Name: "main", Ahead: 1}, Counts: repo.Counts{Staged: 1}}}, {Repository: Repository{Name: "alpha", Path: "/a"}, Snapshot: repo.Snapshot{Branch: repo.Branch{Name: "dev", Behind: 2}, Counts: repo.Counts{Untracked: 2}}}})
	if rows[0].Dirty != 1 || rows[1].Dirty != 2 {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	if got := FilterRows(rows, "dev"); len(got) != 1 || got[0].Repository.Name != "alpha" {
		t.Fatalf("unexpected filter: %#v", got)
	}
	sorted := SortRows(rows, SortName, false)
	if sorted[0].Repository.Name != "alpha" {
		t.Fatalf("unexpected sort: %#v", sorted)
	}
}
