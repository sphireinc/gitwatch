package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/plugins"
	"github.com/sphireinc/git-watch/internal/ui/pluginview"
	"github.com/sphireinc/git-watch/internal/workspace"
)

func TestPluginViewportMouseSelectionUsesVisibleOffset(t *testing.T) {
	m := New()
	m.Width, m.Height = 80, 24
	m.Workspace.Navigate(workspace.Plugins, "Plugins")
	entries := make([]plugins.Entry, 100)
	for i := range entries {
		entries[i].Manifest.Name = fmt.Sprintf("plugin-%03d", i)
	}
	m.Plugins = pluginview.New(entries)
	m.Plugins.Selected = 70
	if !strings.Contains(m.View().Content, "> ◆ plugin-070") {
		t.Fatal("selected plugin must remain visible")
	}
	want := m.Plugins
	if !want.SelectVisibleRow(1, m.Width, m.repositoryViewportLines()) {
		t.Fatal("first visible plugin should be selectable")
	}
	updated, cmd := m.Update(tea.MouseClickMsg{X: 2, Y: 3, Button: tea.MouseLeft})
	got := updated.(Model)
	if cmd != nil || got.Plugins.Selected != want.Selected || got.Plugins.Selected == 0 {
		t.Fatalf("mouse selection %d, want visible offset %d; cmd=%v", got.Plugins.Selected, want.Selected, cmd != nil)
	}
}
