package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/repo"
	"github.com/jusanchez/gitwatch/internal/ui/table"
	"github.com/jusanchez/gitwatch/internal/ui/theme"
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
type DiffReadyMsg struct {
	Path, Text string
	Binary     bool
	Err        error
}

type Model struct {
	State                State
	Width, Height        int
	Focus, Modal, Status string
	Toast                ToastMsg
	Snapshot             repo.Snapshot
	Discovery            git.Discovery
	Files                table.Model
	Theme                theme.Roles
	ctx                  context.Context
	DiffPath, DiffText   string
	DiffBinary           bool
}

func New() Model {
	return Model{State: StateLoading, Focus: "files", Theme: theme.New(theme.Auto, false), ctx: context.Background()}
}
func NewRepository(d git.Discovery) Model { m := New(); m.Discovery = d; return m }
func (m Model) Init() tea.Cmd {
	if m.Discovery.Root == "" {
		return nil
	}
	return m.refresh()
}
func (m Model) refresh() tea.Cmd {
	d := m.Discovery
	return func() tea.Msg {
		s, err := git.Snapshot(context.Background(), d, 0)
		if err != nil {
			return RefreshFinishedMsg{Err: err}
		}
		return SnapshotMsg{Snapshot: s}
	}
}

func (m Model) mutate() tea.Cmd {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	if e.Conflicted {
		return func() tea.Msg {
			return OperationFinishedMsg{Name: "stage", Err: fmt.Errorf("conflicted path requires external resolution")}
		}
	}
	r, path := git.NewRunner(m.Discovery.Root), append([]byte(nil), e.Path...)
	return func() tea.Msg {
		var err error
		if e.Staged && !e.Unstaged {
			_, err = r.Unstage(context.Background(), path)
		} else {
			_, err = r.Stage(context.Background(), path)
		}
		if err != nil {
			return OperationFinishedMsg{Name: "path operation", Err: err}
		}
		s, err := git.Snapshot(context.Background(), m.Discovery, 0)
		if err != nil {
			return RefreshFinishedMsg{Err: err}
		}
		return SnapshotMsg{Snapshot: s}
	}
}

func (m Model) openDiff() tea.Cmd {
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return nil
	}
	e := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	path := append([]byte(nil), e.Path...)
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		d, err := r.Diff(context.Background(), path, e.Staged && !e.Unstaged)
		return DiffReadyMsg{Path: string(path), Text: string(d.Text), Binary: d.Binary, Err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyPressMsg:
		switch v.String() {
		case "q", "ctrl+c":
			m.State = StateShutdown
			return m, tea.Quit
		case "esc":
			if m.Modal != "" {
				m.Modal = ""
				m.State = StateReady
			}
		case "j", "down":
			m.Files.Move(1, m.Height-8)
		case "k", "up":
			m.Files.Move(-1, m.Height-8)
		case "space":
			return m, m.mutate()
		case "enter", "d":
			return m, m.openDiff()
		case "r":
			return m, m.refresh()
		}
	case tea.WindowSizeMsg:
		m.Width, m.Height = v.Width, v.Height
	case SnapshotMsg:
		m.Snapshot = v.Snapshot
		m.Files.SetEntries(v.Snapshot.Entries)
		m.State = StateReady
	case RefreshStartedMsg:
		m.State = StateRefreshing
	case RefreshFinishedMsg:
		if v.Err != nil {
			m.State = StateError
			m.Status = v.Err.Error()
		} else if m.State == StateRefreshing {
			m.State = StateReady
		}
	case WatcherStateMsg:
		if v.Err != nil {
			m.Status = "watcher fallback: " + v.Err.Error()
		}
	case OperationStartedMsg:
		m.State = StateOperationPending
		m.Status = v.Name
	case OperationFinishedMsg:
		if v.Err != nil {
			m.State = StateError
			m.Status = v.Err.Error()
		} else {
			m.State = StateReady
			m.Status = v.Name + " complete"
		}
	case ToastMsg:
		m.Toast = v
	case ModalMsg:
		m.Modal = v.Name
		if v.Open {
			m.State = StateModal
		} else {
			m.State = StateReady
		}
	case FocusMsg:
		m.Focus = v.Pane
	case ShutdownMsg:
		m.State = StateShutdown
	case DiffReadyMsg:
		m.DiffPath, m.DiffText, m.DiffBinary, m.Status = v.Path, v.Text, v.Binary, ""
		if v.Err != nil {
			m.Status = v.Err.Error()
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	name := m.Snapshot.Branch.Name
	if name == "" {
		name = "repository"
	}
	lines := []string{fmt.Sprintf("gitwatch · %s · %s", name, stateName(m.State)), fmt.Sprintf("STAGED %d  MODIFIED %d  UNTRACKED %d  CONFLICTS %d", m.Snapshot.Counts.Staged, m.Snapshot.Counts.Unstaged, m.Snapshot.Counts.Untracked, m.Snapshot.Counts.Conflicted), "──────────────────────────────────────────────────────────────"}
	rows := m.Height - 8
	if rows < 1 {
		rows = 1
	}
	for i := m.Files.Offset; i < len(m.Files.Visible) && i < m.Files.Offset+rows; i++ {
		e := m.Files.Entries[m.Files.Visible[i]]
		prefix := "  "
		if i == m.Files.Selected {
			prefix = "> "
		}
		lines = append(lines, fmt.Sprintf("%s%s %s", prefix, theme.Symbol(repo.StatusLabel(e)), string(e.Path)))
	}
	if len(lines) == 3 {
		lines = append(lines, "  clean worktree")
	}
	if m.Width >= 100 && m.DiffPath != "" {
		lines = append(lines, "", "Selected diff: "+m.DiffPath)
		if m.DiffBinary {
			lines = append(lines, "  [binary file]")
		} else {
			for i, line := range strings.Split(m.DiffText, "\n") {
				if i >= max(1, m.Height-10) {
					break
				}
				lines = append(lines, "  "+line)
			}
		}
	}
	lines = append(lines, "──────────────────────────────────────────────────────────────", "[j/k] move  [space] stage/unstage  [enter/d] diff  [r] refresh  [?] help  [q] quit")
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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
