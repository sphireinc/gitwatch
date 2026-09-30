package repoview

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/health"
	"github.com/sphireinc/git-watch/internal/registry"
)

func TestViewPreservesSelectionAndRendersState(t *testing.T) {
	m := New([]registry.Row{{Repository: registry.Repository{Name: "one", Path: "/one"}, Branch: "main", State: "ready"}, {Repository: registry.Repository{Name: "two", Path: "/two"}, State: "inactive", Dirty: 2, Ahead: 1}})
	m.Move(1)
	m.SetRows([]registry.Row{{Repository: registry.Repository{Name: "two", Path: "/two"}, State: "inactive", Dirty: 2, Ahead: 1}})
	if m.Selected != 0 || !strings.Contains(m.View(), "two") || !strings.Contains(m.View(), "inactive") {
		t.Fatalf("repository view = selected=%d text=%q", m.Selected, m.View())
	}
}

func TestFilterAndSortDashboardRows(t *testing.T) {
	m := New([]registry.Row{{Repository: registry.Repository{Name: "zeta", Path: "/z"}, Dirty: 1}, {Repository: registry.Repository{Name: "alpha", Path: "/a"}, Dirty: 4}})
	m.SetFilter("alpha")
	if len(m.Rows) != 1 || m.Rows[0].Repository.Name != "alpha" {
		t.Fatalf("filtered rows = %#v", m.Rows)
	}
	m.SetFilter("")
	if got := m.CycleSort(); got != registry.SortDirty || m.Rows[0].Repository.Name != "zeta" {
		t.Fatalf("sorted rows = key=%q rows=%#v", got, m.Rows)
	}
}

func TestViewShowsRepositoryWarnings(t *testing.T) {
	m := New([]registry.Row{{Repository: registry.Repository{Name: "repo"}, Warnings: []string{"summary unavailable"}}})
	if !strings.Contains(m.View(), "warnings:1") {
		t.Fatalf("warning count missing: %s", m.View())
	}
}

func TestViewShowsOperationAttentionBadges(t *testing.T) {
	m := New([]registry.Row{{Repository: registry.Repository{Name: "repo"}, Operation: "rebase", Attention: "conflict"}})
	view := m.View()
	if !strings.Contains(view, "op:rebase") || !strings.Contains(view, "attention:conflict") {
		t.Fatalf("operation badges missing: %s", view)
	}
}

func TestViewShowsCachedCIAttentionAndStaleness(t *testing.T) {
	m := New([]registry.Row{{Repository: registry.Repository{Name: "repo"}, ProviderCIState: "failing", ProviderCIStale: true, ProviderCIAttention: "checks"}})
	lines := strings.Split(m.View(), "\n")
	if len(lines) < 3 {
		t.Fatalf("repository view rows = %q", lines)
	}
	stale := "ci:failing(stale)/checks"
	if index := strings.Index(lines[2], stale); index < 0 || index >= 80 {
		t.Fatalf("stale provider state is not prioritized in 80 columns: %q", lines[2])
	}
}

func TestViewShowsMeasuredAutoFetchLatency(t *testing.T) {
	observed := time.Date(2026, 9, 24, 15, 4, 5, 0, time.UTC)
	m := New([]registry.Row{{
		Repository:        registry.Repository{Name: "repo", LastAutoFetchMillis: 1250},
		Health:            health.Summary{Source: "git status", FreshAt: observed},
		RemoteFetchStatus: "fetched",
		RemoteFetchAt:     observed,
		ProviderCIState:   "passing",
	}})
	lines := strings.Split(m.View(), "\n")
	if len(lines) < 3 {
		t.Fatalf("repository view rows = %q", lines)
	}
	latency := "remote-fetch:fetched latency:1250ms"
	if index := strings.Index(lines[1], latency); index < 0 || index >= 80 {
		t.Fatalf("measured remote latency is not prioritized in 80 columns: %q", lines[1])
	}
	for _, detail := range []string{"source:git status observed:15:04:05", "fetch@15:04:05", "ci:passing(fresh)"} {
		if index := strings.Index(lines[2], detail); index < 0 || index >= 80 {
			t.Fatalf("freshness detail %q is not prioritized in 80 columns: %q", detail, lines[2])
		}
	}
	if !strings.Contains(lines[1], "age:") {
		t.Fatalf("fetch age missing: %q", lines[1])
	}
}

func TestFetchAgeHandlesUnknownFutureAndElapsedTimes(t *testing.T) {
	now := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name string
		at   time.Time
		want string
	}{
		{name: "unknown", want: "unknown"},
		{name: "future", at: now.Add(time.Second), want: "clock-skew"},
		{name: "recent", at: now.Add(-30 * time.Second), want: "just now"},
		{name: "minutes", at: now.Add(-3 * time.Minute), want: "3m ago"},
		{name: "hours", at: now.Add(-2 * time.Hour), want: "2h ago"},
		{name: "days", at: now.Add(-48 * time.Hour), want: "2d ago"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := formatFetchAge(test.at, now); got != test.want {
				t.Fatalf("fetch age = %q, want %q", got, test.want)
			}
		})
	}
}

func TestViewBoundsRepositoriesWithoutSplittingPairs(t *testing.T) {
	rows := make([]registry.Row, 256)
	for i := range rows {
		rows[i].Repository = registry.Repository{Name: fmt.Sprintf("repo-%03d", i), Path: fmt.Sprintf("/repo/%03d", i)}
	}
	m := New(rows)
	m.Move(255)
	view := m.View(19)
	lines := strings.Split(view, "\n")
	if len(lines) != 19 {
		t.Fatalf("bounded line count = %d, want 19; view=%q", len(lines), view)
	}
	if !strings.HasPrefix(lines[len(lines)-2], "> repo-255") || !strings.Contains(lines[len(lines)-1], "/repo/255") {
		t.Fatalf("selected repository not visible as a two-line pair: %q", view)
	}
	offset, count := m.VisibleWindow(19)
	if offset != 247 || count != 9 {
		t.Fatalf("visible window = (%d,%d), want (247,9)", offset, count)
	}
}

func TestViewCanDisableAndBoundActivityVisuals(t *testing.T) {
	m := New([]registry.Row{{Repository: registry.Repository{Name: "repo"}, Dirty: 2, Activity: []int{1, 2, 3, 4, 5}}})
	m.SetVisualization(false, 500)
	if m.VisualsEnabled || m.ActivityBuckets != 32 {
		t.Fatalf("visualization settings = enabled=%v buckets=%d", m.VisualsEnabled, m.ActivityBuckets)
	}
	if strings.Contains(m.View(), "heat:") || strings.Contains(m.View(), "activity:") {
		t.Fatalf("disabled visuals remained visible: %s", m.View())
	}
	m.SetVisualization(true, 2)
	if !strings.Contains(m.View(), "activity:") {
		t.Fatalf("enabled visuals missing: %s", m.View())
	}
}
