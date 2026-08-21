package registry

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestRepositoryRowsAllocationBudget(t *testing.T) {
	results := make([]StatusResult, 1_000)
	for i := range results {
		results[i].Repository = Repository{Name: "repo", Path: "/repo"}
		results[i].Snapshot = repo.Snapshot{Branch: repo.Branch{Name: "main"}}
	}
	allocations := testing.AllocsPerRun(1, func() { _ = Rows(results) })
	if allocations > 100 {
		t.Fatalf("1000 repository rows allocations %.0f exceed budget 100", allocations)
	}
}
