package pluginview

import (
	"strings"
	"testing"

	"github.com/jusanchez/gitwatch/internal/plugins"
)

func TestViewShowsDistinctPluginAndCapabilities(t *testing.T) {
	m := New([]plugins.Entry{{Manifest: plugins.Manifest{ID: "one", Name: "One", Version: "1", Capabilities: []plugins.Capability{plugins.CapabilityPanel}}, Enabled: true, Healthy: true}})
	view := m.View()
	for _, want := range []string{"◆ One [enabled]", "capabilities: panel"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
}
