package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestStatusMouseRowHeightsAreViewportBounded(t *testing.T) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}

	m := New()
	m.Files.SetEntries(entries)
	m.Files.Offset = 10000
	got := m.statusFileRowHeights(80, 6)

	if len(got) == 0 || len(got) > 6 {
		t.Fatalf("visible row heights = %d, want 1..6", len(got))
	}
	used := 0
	for _, height := range got {
		if height <= 0 {
			t.Fatalf("row height = %d, want positive", height)
		}
		used += height
	}
	if used > 6 {
		t.Fatalf("visible row height total = %d, want <= 6", used)
	}
}

func TestStatusTreeMouseRowHeightsAreViewportBounded(t *testing.T) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}

	m := New()
	m.Files.SetEntries(entries)
	m.rebuildStatusFileTree()
	m.FileTree.Offset = 10000
	got := m.statusTreeRowHeights(80, 6)

	if len(got) == 0 || len(got) > 6 {
		t.Fatalf("visible tree row heights = %d, want 1..6", len(got))
	}
	used := 0
	for _, height := range got {
		if height <= 0 {
			t.Fatalf("tree row height = %d, want positive", height)
		}
		used += height
	}
	if used > 6 {
		t.Fatalf("visible tree row height total = %d, want <= 6", used)
	}
}

func TestStatusRenderingPreservesLogicalViewportOffset(t *testing.T) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}

	m := New()
	m.Files.SetEntries(entries)
	m.Files.Offset = 10000
	view := m.statusFileLines(80, 6)
	if !strings.Contains(strings.Join(view, "\n"), "generated/10000.txt") {
		t.Fatalf("viewport omitted offset row: %v", view)
	}
	if strings.Contains(strings.Join(view, "\n"), "generated/09999.txt") {
		t.Fatalf("viewport rendered a row before offset: %v", view)
	}
}

func TestStatusMouseRowHeightsAllocationBudget(t *testing.T) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}
	m := New()
	m.Files.SetEntries(entries)
	m.Files.Offset = 10000
	allocations := testing.AllocsPerRun(10, func() {
		_ = m.statusFileRowHeights(80, 22)
	})
	if allocations > 1000 {
		t.Fatalf("bounded status row-height allocations = %.0f, want <= 1000", allocations)
	}
}

func TestStatusMouseRowHeightsScaleAllocationBudget(t *testing.T) {
	for _, size := range []int{1000, 10000, 50000} {
		t.Run(fmt.Sprintf("%d-entries", size), func(t *testing.T) {
			m := statusModelWithEntries(size)
			allocations := testing.AllocsPerRun(5, func() {
				_ = m.statusFileRowHeights(80, 22)
			})
			if allocations > 1000 {
				t.Fatalf("bounded status row-height allocations for %d entries = %.0f, want <= 1000", size, allocations)
			}
		})
	}
}

func BenchmarkStatusMouseRowHeights14953(b *testing.B) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}
	m := New()
	m.Files.SetEntries(entries)
	m.Files.Offset = 10000
	b.Run("bounded-viewport", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = m.statusFileRowHeights(80, 22)
		}
	})
	b.Run("full-scan-baseline", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = fullStatusMouseRowHeightsBaseline(m, 80)
		}
	})
}

func BenchmarkStatusMouseRowHeightsScale(b *testing.B) {
	for _, size := range []int{1000, 10000, 50000} {
		b.Run(fmt.Sprintf("%d-entries", size), func(b *testing.B) {
			m := statusModelWithEntries(size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				_ = m.statusFileRowHeights(80, 22)
			}
		})
	}
}

func statusModelWithEntries(size int) Model {
	entries := make([]repo.Entry, size)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}
	m := New()
	m.Files.SetEntries(entries)
	m.Files.Offset = max(0, size-1000)
	return m
}

// fullStatusMouseRowHeightsBaseline preserves the pre-virtualization work for
// an apples-to-apples benchmark. It intentionally is not used by production
// code or tests that define behavior.
func fullStatusMouseRowHeightsBaseline(m Model, width int) []int {
	heights := make([]int, 0, len(m.Files.Visible)-m.Files.Offset)
	for index := m.Files.Offset; index < len(m.Files.Visible); index++ {
		entry := m.Files.Entries[m.Files.Visible[index]]
		heights = append(heights, len(fitSafeDisplayLines(m.statusFileText(entry, index == m.Files.Selected), width)))
	}
	return heights
}
