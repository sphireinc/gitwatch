// Package pluginview renders discovered plugins and their capability state.
package pluginview

import (
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/plugins"
)

// Model stores the selection state for the plugin view.
type Model struct {
	Entries  []plugins.Entry
	Selected int
}

// New creates a plugin view model with entries selected at the first row.
func New(entries []plugins.Entry) Model {
	return Model{Entries: append([]plugins.Entry(nil), entries...)}
}

// SetEntries replaces rows while keeping the selection within the new bounds.
func (m *Model) SetEntries(entries []plugins.Entry) {
	m.Entries = append([]plugins.Entry(nil), entries...)
	if m.Selected >= len(m.Entries) {
		m.Selected = max(0, len(m.Entries)-1)
	}
}

// Move shifts the selected row by delta and clamps it to the available rows.
func (m *Model) Move(delta int) {
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Entries) {
		m.Selected = max(0, len(m.Entries)-1)
	}
}

// View renders plugin names, capabilities, and health state.
func (m Model) View() string {
	lines := []string{"Plugins"}
	for i, entry := range m.Entries {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		name := entry.Manifest.Name
		if name == "" {
			name = "invalid manifest"
		}
		state := "disabled"
		if entry.Enabled {
			state = "enabled"
		}
		if !entry.Healthy {
			state = "error"
		}
		lines = append(lines, fmt.Sprintf("%s◆ %s [%s]", prefix, platform.SafeText(name), state))
		if entry.Manifest.ID != "" {
			lines = append(lines, "    "+platform.SafeText(entry.Manifest.ID)+" v"+platform.SafeText(entry.Manifest.Version))
		}
		if len(entry.Manifest.Capabilities) > 0 {
			lines = append(lines, "    capabilities: "+platform.SafeText(strings.Join(capabilityNames(entry.Manifest.Capabilities), ", ")))
			permission := "none"
			if entry.Enabled {
				permission = "all declared capabilities"
			}
			lines = append(lines, "    permissions: "+permission)
		}
		if len(entry.Commands) > 0 || len(entry.Panels) > 0 || len(entry.Widgets) > 0 {
			lines = append(lines, fmt.Sprintf("    extensions: commands:%d panels:%d widgets:%d", len(entry.Commands), len(entry.Panels), len(entry.Widgets)))
		}
		if entry.Error != "" {
			lines = append(lines, "    error: "+platform.SafeText(entry.Error))
		}
	}
	if len(m.Entries) == 0 {
		lines = append(lines, "  No plugins discovered")
	}
	return strings.Join(lines, "\n")
}

func capabilityNames(values []plugins.Capability) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = string(value)
	}
	return out
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
