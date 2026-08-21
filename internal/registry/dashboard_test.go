package registry

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestRowsFilterAndSort(t *testing.T) {
	rows := Rows([]StatusResult{{Repository: Repository{Name: "zeta", Path: "/z"}, Stashes: 2, Remotes: 1, Snapshot: repo.Snapshot{Branch: repo.Branch{Name: "main", Ahead: 1}, Counts: repo.Counts{Staged: 1}}}, {Repository: Repository{Name: "alpha", Path: "/a"}, Snapshot: repo.Snapshot{Branch: repo.Branch{Name: "dev", Behind: 2}, Counts: repo.Counts{Untracked: 2}}}})
	if rows[0].Dirty != 1 || rows[1].Dirty != 2 {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	if rows[0].Stashes != 2 {
		t.Fatalf("stash summary = %d", rows[0].Stashes)
	}
	if rows[0].Remotes != 1 {
		t.Fatalf("remote summary = %d", rows[0].Remotes)
	}
	if got := FilterRows(rows, "dev"); len(got) != 1 || got[0].Repository.Name != "alpha" {
		t.Fatalf("unexpected filter: %#v", got)
	}
	sorted := SortRows(rows, SortName, false)
	if sorted[0].Repository.Name != "alpha" {
		t.Fatalf("unexpected sort: %#v", sorted)
	}
}

func TestRowsPreserveWarnings(t *testing.T) {
	rows := Rows([]StatusResult{{Repository: Repository{Name: "repo"}, Warnings: []string{"slow stash"}}})
	if len(rows) != 1 || len(rows[0].Warnings) != 1 {
		t.Fatalf("warnings = %#v", rows)
	}
}
