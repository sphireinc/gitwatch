package pluginview

import (
	"fmt"
	"strings"

	"github.com/jusanchez/gitwatch/internal/platform"
	"github.com/jusanchez/gitwatch/internal/plugins"
)

type Model struct {
	Entries  []plugins.Entry
	Selected int
}

func New(entries []plugins.Entry) Model {
	return Model{Entries: append([]plugins.Entry(nil), entries...)}
}
func (m *Model) SetEntries(entries []plugins.Entry) {
	m.Entries = append([]plugins.Entry(nil), entries...)
	if m.Selected >= len(m.Entries) {
		m.Selected = max(0, len(m.Entries)-1)
	}
}
func (m *Model) Move(delta int) {
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Entries) {
		m.Selected = max(0, len(m.Entries)-1)
	}
}

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
