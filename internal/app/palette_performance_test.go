package app

import (
	"fmt"
	"testing"

	"github.com/sphireinc/git-watch/internal/commands"
	"github.com/sphireinc/git-watch/internal/registry"
	"github.com/sphireinc/git-watch/internal/ui/repoview"
)

func BenchmarkCommandPalette50Repositories(b *testing.B) {
	rows := make([]registry.Row, 50)
	for index := range rows {
		rows[index] = registry.Row{Repository: registry.Repository{Name: fmt.Sprintf("repository-%02d", index), Path: fmt.Sprintf("/workspace/repository-%02d", index)}}
	}
	m := New()
	m.Discovery.Root = "/workspace/repository-00"
	m.Repositories = repoview.New(rows)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		results := commands.Search(m.paletteActions(), "repo: repository-49")
		if len(results) != 1 || results[0].ID != "repository_open_49" {
			b.Fatalf("palette search = %#v", results)
		}
	}
}

func TestCommandPalette50RepositoriesAllocationBudget(t *testing.T) {
	rows := make([]registry.Row, 50)
	for index := range rows {
		rows[index] = registry.Row{Repository: registry.Repository{Name: fmt.Sprintf("repository-%02d", index), Path: fmt.Sprintf("/workspace/repository-%02d", index)}}
	}
	m := New()
	m.Repositories = repoview.New(rows)
	allocations := testing.AllocsPerRun(10, func() {
		results := commands.Search(m.paletteActions(), "repo: repository-49")
		if len(results) != 1 {
			t.Fatalf("palette search returned %d results", len(results))
		}
	})
	if allocations > 2500 {
		t.Fatalf("50-repository palette allocations = %.0f, want <= 2500", allocations)
	}
}
