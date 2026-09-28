package commands

import (
	"fmt"
	"testing"
)

const (
	paletteBenchmarkRepositories         = 50
	paletteBenchmarkEntriesPerRepository = 100
)

func benchmarkLoadedPaletteActions() []Action {
	actions := make([]Action, 0, paletteBenchmarkRepositories*paletteBenchmarkEntriesPerRepository)
	for repository := range paletteBenchmarkRepositories {
		for entry := range paletteBenchmarkEntriesPerRepository {
			actions = append(actions, Action{
				ID:       fmt.Sprintf("commit_%02d_%03d", repository, entry),
				Label:    fmt.Sprintf("repository-%02d commit-%03d", repository, entry),
				Category: "commit",
				Enabled:  true,
			})
		}
	}
	return actions
}

func BenchmarkSearch5000LoadedActions(b *testing.B) {
	actions := benchmarkLoadedPaletteActions()
	for _, benchmark := range []struct {
		name  string
		query string
	}{
		{name: "rare_query", query: "category:commit repository-49 commit-099"},
		{name: "all_results", query: ""},
	} {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				results := Search(actions, benchmark.query)
				if len(results) == 0 {
					b.Fatal("palette search returned no results")
				}
			}
		})
	}
}

func TestSearch5000LoadedActionsAllocationBudget(t *testing.T) {
	actions := benchmarkLoadedPaletteActions()
	for _, query := range []string{"category:commit repository-49 commit-099", ""} {
		allocations := testing.AllocsPerRun(10, func() {
			if results := Search(actions, query); len(results) == 0 {
				t.Fatalf("Search(%q) returned no results", query)
			}
		})
		if allocations > 1000 {
			t.Errorf("Search(%q) allocations = %.0f, want <= 1000", query, allocations)
		}
	}
}
