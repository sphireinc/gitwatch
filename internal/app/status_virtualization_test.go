package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/ui/layout"
	"github.com/sphireinc/git-watch/internal/ui/theme"
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

func TestStatusVirtualizationPreservesOffscreenSelectionAndFiltering(t *testing.T) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}

	m := New()
	m.Files.SetEntries(entries)
	m.Files.Selected = 12000
	m.Files.Offset = 12000
	m.moveStatusFiles(1)
	if got := m.Files.SelectedPath(); got != "generated/12001.txt" {
		t.Fatalf("offscreen selection moved to %q", got)
	}
	page := m.statusRowCount()
	m.moveStatusFiles(page)
	if got, want := m.Files.SelectedPath(), fmt.Sprintf("generated/%05d.txt", 12001+page); got != want {
		t.Fatalf("offscreen page movement selected %q, want %q", got, want)
	}
	m.Files.SetFilter("generated/14952")
	if len(m.Files.Visible) != 1 || m.Files.Selected != 0 || m.Files.SelectedPath() != "generated/14952.txt" {
		t.Fatalf("filtered offscreen selection = visible=%d selected=%d path=%q", len(m.Files.Visible), m.Files.Selected, m.Files.SelectedPath())
	}
}

func TestStatusSnapshotRefreshKeepsOffscreenSelectionVisible(t *testing.T) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}
	m := New()
	m.Width, m.Height = 80, 24
	m.Snapshot.Entries = entries
	m.Files.SetEntries(entries)
	statusLayout := m.statusLayout()
	fileWidth := statusLayout.Files.Width
	if statusLayout.Mode == layout.Wide {
		fileWidth = max(1, fileWidth-1)
	}
	viewportRows := m.statusRowCount()
	m.Files.Move(len(entries)-1, viewportRows)
	selected := m.Files.SelectedPath()

	refreshed := append([]repo.Entry{{Path: repo.Path("aaa.txt"), Untracked: true}}, entries...)
	m.applySnapshot(repo.Snapshot{Entries: refreshed})
	if got := m.Files.SelectedPath(); got != selected {
		t.Fatalf("refresh changed selected path from %q to %q", selected, got)
	}
	if m.Files.Selected < m.Files.Offset || m.Files.Selected >= m.Files.Offset+viewportRows {
		t.Fatalf("refreshed selection %d is outside logical viewport [%d,%d)", m.Files.Selected, m.Files.Offset, m.Files.Offset+viewportRows)
	}
	if rendered := strings.Join(m.statusFileLines(fileWidth, statusLayout.Files.Height), "\n"); !strings.Contains(rendered, selected) {
		t.Fatalf("refreshed viewport omitted selected path %q: %s", selected, rendered)
	}
}

func TestStatusRestoreConfirmationUsesOffscreenSelection(t *testing.T) {
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Unstaged: true}
	}
	m := New()
	m.Files.SetEntries(entries)
	m.Files.Move(len(entries)-1, 8)
	target := m.Files.SelectedPath()
	m.beginRestore()
	if !m.Restore.Open || m.Restore.Path != target {
		t.Fatalf("restore confirmation = open:%v path:%q, want selected offscreen path %q", m.Restore.Open, m.Restore.Path, target)
	}
}

func TestStatusVirtualizationAt80x24HonorsNoColorAndMotionModes(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	entries := make([]repo.Entry, 14953)
	for index := range entries {
		entries[index] = repo.Entry{Path: repo.Path(fmt.Sprintf("generated/%05d.txt", index)), Untracked: true}
	}

	m := New()
	m.Width, m.Height = 80, 24
	m.Theme = theme.New(theme.Dark, false)
	m.Files.SetEntries(entries)
	m.Files.Selected = 14952
	m.Files.Offset = 14940
	for _, motion := range []Motion{MotionFull, MotionReduced, MotionOff} {
		m.Motion = motion
		content := m.View().Content
		lines := strings.Split(content, "\n")
		if len(lines) != m.Height {
			t.Fatalf("motion %q rendered %d lines at 80x24, want %d", motion, len(lines), m.Height)
		}
		if strings.Contains(content, "\x1b[") {
			t.Fatalf("motion %q emitted terminal color under NO_COLOR", motion)
		}
		if !strings.Contains(content, "generated/14952.txt") {
			t.Fatalf("motion %q omitted selected offscreen logical path", motion)
		}
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

func TestStatusRenderingAllocationBudgetScale(t *testing.T) {
	var baseline float64
	for _, size := range []int{14953, 50000} {
		t.Run(fmt.Sprintf("%d-entries", size), func(t *testing.T) {
			m := statusModelWithEntries(size)
			m.Width, m.Height = 80, 24
			m.StatusOverscan = 4
			m.Files.Offset = 0
			allocations := testing.AllocsPerRun(5, func() {
				_ = m.statusView()
			})
			if allocations > 1500 {
				t.Fatalf("status render allocations for %d entries = %.0f, want <= 1500", size, allocations)
			}
			if baseline == 0 {
				baseline = allocations
			} else if allocations > baseline+64 {
				t.Fatalf("status render allocations grew from %.0f at 14,953 entries to %.0f at %d entries; want growth <= 64", baseline, allocations, size)
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

func BenchmarkStatusFileLines14953(b *testing.B) {
	m := statusModelWithEntries(14953)
	m.StatusOverscan = 4
	m.Files.Offset = 0
	b.Run("bounded-viewport", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = m.statusFileLines(80, 22)
		}
	})
	b.Run("full-scan-baseline", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			_ = fullStatusFileLinesBaseline(m, 80, 22)
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

func fullStatusFileLinesBaseline(m Model, width, height int) []string {
	lines := make([]string, 0, len(m.Files.Visible)-m.Files.Offset)
	for index := m.Files.Offset; index < len(m.Files.Visible); index++ {
		entry := m.Files.Entries[m.Files.Visible[index]]
		lines = append(lines, fitSafeDisplayLines(m.statusFileText(entry, index == m.Files.Selected), width)...)
	}
	return padStatusPanel(lines, width, height)
}
