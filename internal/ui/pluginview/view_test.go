package pluginview

import (
	"strings"
	"testing"

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
