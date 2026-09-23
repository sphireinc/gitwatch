package repoview

import (
	"strings"
	"testing"

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
	view := m.View()
	for _, want := range []string{"ci:failing(stale)/checks"} {
		if !strings.Contains(view, want) {
			t.Fatalf("cached CI badge missing %q: %s", want, view)
		}
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
