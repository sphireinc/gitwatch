package app

import (
	"charm.land/bubbletea/v2"
)

// Model is the minimal full-screen shell. Feature state is added by later tasks.
type Model struct {
	width  int
	height int
}

func New() Model { return Model{} }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}
	return m, nil
}

func (m Model) View() tea.View {
	content := "gitwatch\n\nRepository dashboard loading...\n\nq quit · ? help"
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
