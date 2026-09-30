// Package pluginview renders discovered plugins and their capability state.
package pluginview

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/plugins"
	"github.com/sphireinc/git-watch/pkg/plugin"
)

const (
	defaultViewportWidth  = 80
	defaultViewportHeight = 24
	maxViewportWidth      = 240
	maxViewportHeight     = 200
	maxTableColumns       = 4
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
	return m.ViewWithSize(defaultViewportWidth, defaultViewportHeight)
}

// ViewWithSize renders bounded, schema-defined plugin data inside a terminal
// viewport. Width and height are capped to keep malformed callers from
// creating an unbounded render workload.
func (m Model) ViewWithSize(width, height int) string {
	width, height = normalizeViewport(width, height)
	lines := []string{"Plugins"}
	if len(m.Entries) == 0 {
		if height > 1 {
			lines = append(lines, "  No plugins discovered")
		}
		return strings.Join(fitLines(lines, width, height), "\n")
	}

	selected := min(max(m.Selected, 0), len(m.Entries)-1)
	listRows := visibleListRows(len(m.Entries), height)
	start := visibleStart(selected, len(m.Entries), listRows)
	for i := start; i < start+listRows && len(lines) < height; i++ {
		entry := m.Entries[i]
		prefix := "  "
		if i == selected {
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
		lines = append(lines, fmt.Sprintf("%s◆ %s [%s]", prefix, safeSingleLine(name), state))
	}

	truncated := false
	entry := m.Entries[selected]
	if height-len(lines) >= 3 {
		lines = append(lines, "")
		appendLine := func(line string) bool {
			if len(lines) >= height {
				truncated = true
				return false
			}
			lines = append(lines, line)
			return true
		}
		if entry.Manifest.ID != "" {
			appendLine("  " + safeSingleLine(entry.Manifest.ID) + " v" + safeSingleLine(entry.Manifest.Version))
		}
		if len(entry.Manifest.Capabilities) > 0 {
			appendLine("  capabilities: " + safeSingleLine(strings.Join(capabilityNames(entry.Manifest.Capabilities), ", ")))
			permission := "none"
			if entry.Enabled {
				permission = "all declared capabilities"
			}
			appendLine("  permissions: " + permission)
		}
		if len(entry.Commands) > 0 || len(entry.Panels) > 0 || len(entry.Widgets) > 0 {
			appendLine(fmt.Sprintf("  extensions: commands:%d panels:%d widgets:%d", len(entry.Commands), len(entry.Panels), len(entry.Widgets)))
		}
		if len(entry.Contributions) > 0 {
			for _, contribution := range entry.Contributions {
				if !appendLine("    " + safeSingleLine(contribution.Kind) + ": " + safeSingleLine(contribution.Title)) {
					break
				}
				if contribution.Description != "" && !appendLine("      "+safeSingleLine(contribution.Description)) {
					break
				}
				if contribution.Action != nil && !appendLine("      action: "+safeSingleLine(contribution.Action.Title)+" · "+safeSingleLine(contribution.Action.Provider)) {
					break
				}
				lines, truncated = appendContributionData(lines, contribution, width, height)
				if truncated || len(lines) >= height {
					break
				}
			}
		}
		if entry.Error != "" && len(lines) < height {
			appendLine("  error: " + safeSingleLine(entry.Error))
		}
	}
	if len(lines) > height {
		lines = lines[:height]
		truncated = true
	}
	if truncated && len(lines) > 0 {
		lines[len(lines)-1] = "… viewport truncated"
	}
	return strings.Join(fitLines(lines, width, height), "\n")
}

// SelectVisibleRow maps a zero-based plugin-content row to the matching entry.
// Content row zero is the heading; plugin rows start at content row one.
func (m *Model) SelectVisibleRow(contentRow, width, height int) bool {
	if m == nil || len(m.Entries) == 0 || contentRow < 1 {
		return false
	}
	_, height = normalizeViewport(width, height)
	selected := min(max(m.Selected, 0), len(m.Entries)-1)
	listRows := visibleListRows(len(m.Entries), height)
	start := visibleStart(selected, len(m.Entries), listRows)
	if contentRow > listRows {
		return false
	}
	m.Selected = start + contentRow - 1
	return m.Selected < len(m.Entries)
}

func normalizeViewport(width, height int) (int, int) {
	if width <= 0 {
		width = defaultViewportWidth
	}
	if height <= 0 {
		height = defaultViewportHeight
	}
	return min(width, maxViewportWidth), min(height, maxViewportHeight)
}

func visibleStart(selected, entries, rows int) int {
	if rows <= 0 {
		return 0
	}
	return min(max(selected-rows/2, 0), entries-rows)
}

func visibleListRows(entries, height int) int {
	if entries == 0 || height <= 1 {
		return 0
	}
	return min(entries, max(1, (height-1)/3))
}

func fitLines(lines []string, width, height int) []string {
	if len(lines) > height {
		lines = lines[:height]
	}
	for i := range lines {
		lines[i] = truncateLine(lines[i], width)
	}
	return lines
}

func appendContributionData(lines []string, contribution plugin.Contribution, viewportWidth, height int) ([]string, bool) {
	truncated := false
	appendLine := func(line string) bool {
		if len(lines) >= height {
			truncated = true
			return false
		}
		lines = append(lines, line)
		return true
	}
	switch contribution.Kind {
	case "table":
		columns := contribution.Columns
		if len(columns) == 0 && len(contribution.Rows) > 0 {
			for id := range contribution.Rows[0] {
				columns = append(columns, plugin.TableColumn{ID: id, Title: id})
			}
			sort.Slice(columns, func(i, j int) bool { return columns[i].ID < columns[j].ID })
		}
		if len(columns) == 0 {
			return lines, false
		}
		indent := "        "
		cellBudget := max(1, viewportWidth-runewidth.StringWidth(indent))
		columnLimit := min(len(columns), maxTableColumns, max(1, (cellBudget+3)/10))
		shown := columns[:columnLimit]
		cellWidth := max(1, (cellBudget-(columnLimit-1)*3)/columnLimit)
		appendCells := func(values []string) string {
			cells := make([]string, len(shown))
			for i, value := range values {
				cells[i] = runewidth.Truncate(safeSingleLine(value), cellWidth, "…")
			}
			return indent + strings.Join(cells, " │ ")
		}
		titles := make([]string, len(shown))
		for i, column := range shown {
			titles[i] = column.Title
		}
		if !appendLine(appendCells(titles)) {
			return lines, true
		}
		for _, row := range contribution.Rows {
			values := make([]string, len(shown))
			for i, column := range shown {
				values[i] = row[column.ID]
			}
			if !appendLine(appendCells(values)) {
				return lines, true
			}
		}
		if columnLimit < len(columns) {
			appendLine(fmt.Sprintf("        … %d more columns", len(columns)-columnLimit))
		}
	case "detail":
		keys := make([]string, 0, len(contribution.Fields))
		for key := range contribution.Fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !appendLine("        " + safeSingleLine(key) + ": " + safeSingleLine(contribution.Fields[key])) {
				return lines, true
			}
		}
	}
	return lines, truncated
}

func safeSingleLine(value string) string {
	value = platform.SafeText(value)
	value = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(value)
	return value
}

func truncateLine(value string, width int) string {
	return runewidth.Truncate(safeSingleLine(value), width, "…")
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
