package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/branches"
	"github.com/jusanchez/gitwatch/internal/config"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/history"
	"github.com/jusanchez/gitwatch/internal/repo"
	"github.com/jusanchez/gitwatch/internal/stash"
	"github.com/jusanchez/gitwatch/internal/ui/branchview"
	"github.com/jusanchez/gitwatch/internal/ui/historyview"
	"github.com/jusanchez/gitwatch/internal/ui/layout"
	uimouse "github.com/jusanchez/gitwatch/internal/ui/mouse"
	"github.com/jusanchez/gitwatch/internal/ui/stashview"
	"github.com/jusanchez/gitwatch/internal/ui/table"
	"github.com/jusanchez/gitwatch/internal/ui/theme"
	"github.com/jusanchez/gitwatch/internal/workspace"
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
type BranchesReadyMsg struct {
	Entries []branches.Branch
	Err     error
}
type StashesReadyMsg struct {
	Entries []stash.Entry
	Err     error
}
type HistoryReadyMsg struct {
	Commits []history.Commit
	Err     error
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
	RefreshInterval      time.Duration
	DiffPath, DiffText   string
	DiffBinary           bool
	Workspace            *workspace.Model
	Branches             branchview.Model
	Stashes              stashview.Model
	History              historyview.Model
}

func New() Model {
	return Model{State: StateLoading, Focus: "files", Theme: theme.New(theme.Auto, false), ctx: context.Background(), RefreshInterval: 2 * time.Second, Workspace: workspace.New()}
}
func NewRepository(d git.Discovery) Model { m := New(); m.Discovery = d; return m }
func NewRepositoryWithConfig(d git.Discovery, c config.Config) Model {
	m := NewRepository(d)
	m.RefreshInterval = c.Interval
	m.Theme = theme.New(theme.Name(c.Theme), false)
	return m
}
func (m Model) Init() tea.Cmd {
	if m.Discovery.Root == "" {
		return nil
	}
	return tea.Batch(m.refresh(), m.tick())
}

func (m Model) tick() tea.Cmd {
	interval := m.RefreshInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return tea.Tick(interval, func(t time.Time) tea.Msg { return TickMsg{At: t} })
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

func (m Model) loadBranches() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := branches.List(context.Background(), r)
		return BranchesReadyMsg{Entries: entries, Err: err}
	}
}

func (m Model) loadStashes() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		entries, err := stash.List(context.Background(), r)
		return StashesReadyMsg{Entries: entries, Err: err}
	}
}

func (m Model) loadHistory() tea.Cmd {
	r := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		commits, err := history.LoadLog(context.Background(), r, 100)
		return HistoryReadyMsg{Commits: commits, Err: err}
	}
}

func (m Model) navigate(view workspace.View, label string) tea.Cmd {
	if m.Workspace == nil {
		m.Workspace = workspace.New()
	}
	m.Workspace.Navigate(view, label)
	switch view {
	case workspace.Branches:
		return m.loadBranches()
	case workspace.Stashes:
		return m.loadStashes()
	case workspace.Log:
		return m.loadHistory()
	default:
		return nil
	}
}

func (m Model) currentView() workspace.View {
	if m.Workspace == nil {
		return workspace.Status
	}
	view, _, _, _ := m.Workspace.Snapshot()
	return view
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
			} else if m.currentView() != workspace.Status {
				m.Workspace.Back()
			}
		case "1":
			m.Workspace.Navigate(workspace.Status, "Status")
		case "b":
			return m, m.navigate(workspace.Branches, "Branches")
		case "s":
			return m, m.navigate(workspace.Stashes, "Stashes")
		case "l":
			return m, m.navigate(workspace.Log, "History")
		case "j", "down":
			switch m.currentView() {
			case workspace.Branches:
				m.Branches.Move(1)
			case workspace.Stashes:
				m.Stashes.Move(1)
			case workspace.Log:
				m.History.Move(1)
			default:
				m.Files.Move(1, m.Height-8)
			}
		case "k", "up":
			switch m.currentView() {
			case workspace.Branches:
				m.Branches.Move(-1)
			case workspace.Stashes:
				m.Stashes.Move(-1)
			case workspace.Log:
				m.History.Move(-1)
			default:
				m.Files.Move(-1, m.Height-8)
			}
		case "space":
			return m, m.mutate()
		case "enter", "d":
			return m, m.openDiff()
		case "?":
			m.Modal, m.State = "help", StateModal
		case "r":
			return m, m.refresh()
		}
	case tea.MouseWheelMsg:
		if v.Button == tea.MouseWheelUp {
			m.Files.Move(-1, m.Height-8)
		} else if v.Button == tea.MouseWheelDown {
			m.Files.Move(1, m.Height-8)
		}
	case tea.MouseClickMsg:
		if v.Button == tea.MouseLeft {
			hit := uimouse.HitMap{Files: layout.Rect{X: 0, Y: 3, Width: m.Width, Height: max(1, m.Height-8)}, RowTop: 3, RowHeight: 1, StageX: 0, StageWidth: 3, RowCount: len(m.Files.Visible)}
			action, row, ok := hit.Hit(v.X, v.Y, 0)
			if ok {
				m.Files.Selected = row
				if action == uimouse.ToggleStage {
					return m, m.mutate()
				}
				if action == uimouse.SelectRow {
					return m, m.openDiff()
				}
			}
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
	case TickMsg:
		return m, tea.Batch(m.refresh(), m.tick())
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
	case BranchesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Branches, m.State = branchview.New(v.Entries), StateReady
		}
	case StashesReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.Stashes, m.State = stashview.New(v.Entries), StateReady
		}
	case HistoryReadyMsg:
		if v.Err != nil {
			m.State, m.Status = StateError, v.Err.Error()
		} else {
			m.History, m.State = historyview.New(v.Commits), StateReady
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if view := m.currentView(); view == workspace.Branches || view == workspace.Stashes || view == workspace.Log {
		return m.featureView(view)
	}
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
	if m.Modal == "help" {
		lines = []string{"gitwatch help", "", "↑/↓ or j/k   move selection", "click row     select and open diff", "Space          stage or unstage", "Enter or d     open selected diff", "b              branches", "s              stashes", "l              history", "1              status", "r              refresh", "Esc            close help", "q              quit"}
	}
	lines = append(lines, "──────────────────────────────────────────────────────────────", "[j/k] move  [space] stage/unstage  [enter/d] diff  [r] refresh  [?] help  [q] quit")
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) featureView(view workspace.View) tea.View {
	title, content := "gitwatch", "Loading…"
	if view == workspace.Branches {
		title, content = "gitwatch · branches", m.Branches.View()
	} else if view == workspace.Stashes {
		title, content = "gitwatch · stashes", m.Stashes.View()
	} else if view == workspace.Log {
		title, content = "gitwatch · history", m.History.View()
	}
	lines := []string{title, "", content, "", "──────────────────────────────────────────────────────────────", "[j/k] move  [1] status  [b] branches  [s] stashes  [l] history  [esc] back  [q] quit"}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen, v.MouseMode = true, tea.MouseModeCellMotion
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
