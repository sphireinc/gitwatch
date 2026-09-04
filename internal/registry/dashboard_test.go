package registry

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/sequencer"
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

func TestRowsExposeOperationAndAttentionPriority(t *testing.T) {
	operation, err := sequencer.NewState("/conflicted", 1, sequencer.KindRebase, sequencer.PhaseActive)
	if err != nil {
		t.Fatal(err)
	}
	results := []StatusResult{
		{Repository: Repository{Name: "conflicted", Path: "/conflicted"}, Snapshot: repo.Snapshot{Operation: &operation, Counts: repo.Counts{Conflicted: 1}}},
		{Repository: Repository{Name: "failed", Path: "/failed"}, OperationFailed: true, Snapshot: repo.Snapshot{Counts: repo.Counts{Untracked: 1}}},
		{Repository: Repository{Name: "provider", Path: "/provider"}, ProviderAttention: true, Snapshot: repo.Snapshot{}},
	}
	rows := Rows(results)
	if rows[0].Operation != "rebase" || rows[0].Attention != "conflict" || rows[1].Attention != "operation failed" || rows[2].Attention != "provider stale" {
		t.Fatalf("attention rows = %#v", rows)
	}
	if got := FilterRows(rows, "rebase"); len(got) != 1 || !strings.Contains(got[0].Attention, "conflict") {
		t.Fatalf("operation filter = %#v", got)
	}
}
