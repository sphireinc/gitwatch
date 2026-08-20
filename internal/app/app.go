package app

import (
	"fmt"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/repo"
)

type State uint8

const (
	StateLoading State = iota
	StateReady
	StateRefreshing
	StateOperationPending
	StateError
	StateModal
	StateShutdown
)

type SnapshotMsg struct{ Snapshot repo.Snapshot }
type RefreshStartedMsg struct{}
type RefreshFinishedMsg struct{ Err error }
type WatcherStateMsg struct {
	Mode string
	Err  error
}
type OperationStartedMsg struct{ Name string }
type OperationFinishedMsg struct {
	Name string
	Err  error
}
type TickMsg struct{ At time.Time }
type ToastMsg struct {
	Text  string
	Error bool
}
type ModalMsg struct {
	Open bool
	Name string
}
type FocusMsg struct{ Pane string }
type ShutdownMsg struct{}

type Model struct {
	State    State
	Width    int
	Height   int
	Focus    string
	Modal    string
	Status   string
	Toast    ToastMsg
	Snapshot repo.Snapshot
}

func New() Model              { return Model{State: StateLoading, Focus: "files"} }
func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.State = StateShutdown
			return m, tea.Quit
		}
		if msg.String() == "esc" && m.Modal != "" {
			m.Modal = ""
			m.State = StateReady
		}
	case tea.WindowSizeMsg:
		m.Width, m.Height = msg.Width, msg.Height
	case SnapshotMsg:
		m.Snapshot = msg.Snapshot
		m.State = StateReady
	case RefreshStartedMsg:
		m.State = StateRefreshing
	case RefreshFinishedMsg:
		if msg.Err != nil {
			m.State = StateError
			m.Status = msg.Err.Error()
		} else if m.State == StateRefreshing {
			m.State = StateReady
		}
	case WatcherStateMsg:
		if msg.Err != nil {
			m.Status = "watcher fallback: " + msg.Err.Error()
		}
	case OperationStartedMsg:
		m.State = StateOperationPending
		m.Status = msg.Name
	case OperationFinishedMsg:
		if msg.Err != nil {
			m.State = StateError
			m.Status = msg.Err.Error()
		} else {
			m.State = StateReady
			m.Status = msg.Name + " complete"
		}
	case ToastMsg:
		m.Toast = msg
	case ModalMsg:
		m.Modal = msg.Name
		if msg.Open {
			m.State = StateModal
		} else {
			m.State = StateReady
		}
	case FocusMsg:
		m.Focus = msg.Pane
	case ShutdownMsg:
		m.State = StateShutdown
	}
	return m, nil
}

func (m Model) View() tea.View {
	name := m.Snapshot.Branch.Name
	if name == "" {
		name = "repository"
	}
	content := fmt.Sprintf("gitwatch · %s\n\nstate: %s\nstatus: %s\n\nFiles and diff panes are coming online...\n\nq quit · ? help", name, stateName(m.State), m.Status)
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func stateName(s State) string {
	switch s {
	case StateLoading:
		return "loading"
	case StateReady:
		return "ready"
	case StateRefreshing:
		return "refreshing"
	case StateOperationPending:
		return "operation pending"
	case StateError:
		return "error"
	case StateModal:
		return "modal"
	case StateShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}
