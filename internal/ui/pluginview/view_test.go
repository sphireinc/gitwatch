package pluginview

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/plugins"
	"github.com/sphireinc/git-watch/pkg/plugin"
)

func TestViewShowsDistinctPluginAndCapabilities(t *testing.T) {
	m := New([]plugins.Entry{{Manifest: plugins.Manifest{ID: "one", Name: "One", Version: "1", Capabilities: []plugins.Capability{plugins.CapabilityPanel}}, Enabled: true, Healthy: true}})
	view := m.View()
	for _, want := range []string{"◆ One [enabled]", "capabilities: panel", "permissions: all declared capabilities"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
}

func TestViewShowsExtensionSurfaceCountsAndPermissionRevocation(t *testing.T) {
	entry := plugins.Entry{Manifest: plugins.Manifest{ID: "one", Name: "One", Version: "1", Capabilities: []plugins.Capability{plugins.CapabilityCommand}}, Enabled: false, Healthy: true}
	entry.Commands = []plugin.CommandSpec{{ID: "refresh", Title: "Refresh"}}
	view := New([]plugins.Entry{entry}).View()
	for _, want := range []string{"permissions: none", "extensions: commands:1 panels:0 widgets:0"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
}

func TestViewShowsSchemaDefinedContributions(t *testing.T) {
	entry := plugins.Entry{Manifest: plugins.Manifest{ID: "one", Name: "One", Version: "2"}, Enabled: true, Healthy: true}
	entry.Contributions = []plugin.Contribution{{
		SchemaVersion: plugin.APIVersion2, Kind: "repository_metadata", Title: "Repository health", ReadOnly: true,
		Action: &plugin.ActionSpec{ID: "github", Title: "Open GitHub metadata", Provider: plugin.ActionProviderGitHubRepository, ReadOnly: true},
	}}
	view := New([]plugins.Entry{entry}).View()
	for _, want := range []string{"repository_metadata: Repository health", "action: Open GitHub metadata · github.repository"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
}

func TestViewRendersBoundedTableAndDetailData(t *testing.T) {
	entry := plugins.Entry{Manifest: plugins.Manifest{ID: "one", Name: "One"}, Enabled: true, Healthy: true}
	entry.Contributions = []plugin.Contribution{
		{
			SchemaVersion: plugin.APIVersion2, Kind: "table", Title: "Recent builds", ReadOnly: true,
			Columns: []plugin.TableColumn{{ID: "branch", Title: "Branch"}, {ID: "result", Title: "Result"}},
			Rows:    []plugin.TableRow{{"branch": "main", "result": "passed"}},
		},
		{
			SchemaVersion: plugin.APIVersion2, Kind: "detail", Title: "Repository facts", ReadOnly: true,
			Fields: map[string]string{"zeta": "last", "alpha": "first"},
		},
	}

	view := New([]plugins.Entry{entry}).ViewWithSize(80, 24)
	for _, want := range []string{"Branch", "Result", "main", "passed", "alpha: first", "zeta: last"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestViewSanitizesAndBoundsContributionTextToViewport(t *testing.T) {
	entry := plugins.Entry{Manifest: plugins.Manifest{ID: "one", Name: "One"}, Enabled: true, Healthy: true}
	entry.Contributions = []plugin.Contribution{{
		SchemaVersion: plugin.APIVersion2, Kind: "detail", Title: "hostile\x1b[31m title", ReadOnly: true,
		Fields: map[string]string{"note": "wide 界 value\nsecond line"},
	}}
	view := New([]plugins.Entry{entry}).ViewWithSize(50, 7)
	if strings.Contains(view, "\x1b") || strings.Contains(view, "\nsecond line") {
		t.Fatalf("plugin text escaped sanitization/line boundary: %q", view)
	}
	lines := strings.Split(view, "\n")
	if len(lines) > 7 {
		t.Fatalf("view should be clipped to seven lines: %q", view)
	}
	for i, line := range lines {
		if got := runewidth.StringWidth(line); got > 50 {
			t.Errorf("line %d width = %d, want <= 50: %q", i, got, line)
		}
	}
	if !strings.Contains(view, "hostile�[31m title") {
		t.Fatalf("unsafe title was not visibly replaced: %q", view)
	}
}

func TestViewSelectionWindowFollowsSelectedAndMapsVisibleRows(t *testing.T) {
	entries := make([]plugins.Entry, 10)
	for i := range entries {
		entries[i] = plugins.Entry{
			Manifest: plugins.Manifest{ID: "plugin-id", Name: fmt.Sprintf("Plugin %02d", i)},
			Enabled:  true, Healthy: true,
			Contributions: []plugin.Contribution{{
				SchemaVersion: plugin.APIVersion2, Kind: "detail", Title: "Repository", ReadOnly: true,
				Fields: map[string]string{"repository": fmt.Sprintf("repo-%d", i)},
			}},
		}
	}
	m := New(entries)
	m.Selected = 8
	view := m.ViewWithSize(80, 12)
	if !strings.Contains(view, "Plugin 08") || !strings.Contains(view, "repo-8") || strings.Contains(view, "repo-0") {
		t.Fatalf("viewport did not follow selected plugin detail:\n%s", view)
	}
	if !m.SelectVisibleRow(3, 80, 12) || m.Selected != 9 {
		t.Fatalf("visible third plugin row should select entry 9, selected=%d", m.Selected)
	}
	if m.SelectVisibleRow(4, 80, 12) {
		t.Fatal("row outside the visible plugin list should not select an entry")
	}
	view = m.ViewWithSize(80, 12)
	if !strings.Contains(view, "repo-9") || strings.Contains(view, "repo-0") {
		t.Fatalf("selection update did not move selected details:\n%s", view)
	}
}

func TestViewStopsRenderingRowsAtViewportBudget(t *testing.T) {
	rows := make([]plugin.TableRow, plugin.MaxContributionRows)
	for i := range rows {
		rows[i] = plugin.TableRow{"name": fmt.Sprintf("item-%03d", i)}
	}
	entry := plugins.Entry{
		Manifest: plugins.Manifest{Name: "bounded"}, Enabled: true, Healthy: true,
		Contributions: []plugin.Contribution{{
			SchemaVersion: plugin.APIVersion2, Kind: "table", Title: "Rows", ReadOnly: true,
			Columns: []plugin.TableColumn{{ID: "name", Title: "Name"}}, Rows: rows,
		}},
	}
	view := New([]plugins.Entry{entry}).ViewWithSize(32, 9)
	lines := strings.Split(view, "\n")
	if len(lines) != 9 || lines[8] != "… viewport truncated" {
		t.Fatalf("expected nine lines with a truncation marker, got %d:\n%s", len(lines), view)
	}
	if !strings.Contains(view, "item-000") || strings.Contains(view, "item-010") {
		t.Fatalf("row rendering should stop at the viewport boundary:\n%s", view)
	}
}

func TestViewCapsRequestedViewport(t *testing.T) {
	view := New([]plugins.Entry{{Manifest: plugins.Manifest{Name: strings.Repeat("界", 400)}, Enabled: true, Healthy: true}}).ViewWithSize(1000, 1000)
	lines := strings.Split(view, "\n")
	if len(lines) > maxViewportHeight {
		t.Fatalf("line count = %d, exceeds cap %d", len(lines), maxViewportHeight)
	}
	for i, line := range lines {
		if got := runewidth.StringWidth(line); got > maxViewportWidth {
			t.Errorf("line %d width = %d, exceeds cap %d", i, got, maxViewportWidth)
		}
	}
}
