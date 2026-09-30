package table

import (
	"fmt"
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
)

func BenchmarkTable10KFilter(b *testing.B) {
	benchmarkTableFilterSize(b, 10_000)
}

func BenchmarkTableFilterScale(b *testing.B) {
	for _, size := range []int{10_000, 50_000} {
		b.Run(fmt.Sprintf("%d-entries", size), func(b *testing.B) {
			benchmarkTableFilterSize(b, size)
		})
	}
}

func benchmarkTableFilterSize(b *testing.B, size int) {
	b.Helper()
	entries := tableEntries(size)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := New(entries)
		m.SetFilter("9999")
	}
}

func TestTableFilterAllocationBudgetScale(t *testing.T) {
	var baseline float64
	for _, size := range []int{10_000, 50_000} {
		t.Run(fmt.Sprintf("%d-entries", size), func(t *testing.T) {
			entries := tableEntries(size)
			allocations := testing.AllocsPerRun(1, func() {
				m := New(entries)
				m.SetFilter("9999")
			})
			if allocations > 64 {
				t.Fatalf("filter allocations at %d entries = %.0f, want <= 64 independent of registry size", size, allocations)
			}
			if baseline == 0 {
				baseline = allocations
			} else if allocations > baseline+16 {
				t.Fatalf("filter allocations grew from %.0f at 10k entries to %.0f at %d entries, want growth <= 16", baseline, allocations, size)
			}
		})
	}
}

func tableEntries(size int) []repo.Entry {
	entries := make([]repo.Entry, size)
	for i := range entries {
		entries[i] = repo.Entry{Path: repo.Path(fmt.Sprintf("dir/%05d.go", i))}
	}
	return entries
}
