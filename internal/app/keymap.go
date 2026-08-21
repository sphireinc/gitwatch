package app

type Binding struct {
	Key, Action, Description string
	Destructive              bool
}

var DefaultBindings = []Binding{
	{Key: "↑/↓, j/k", Action: "move", Description: "move selection"},
	{Key: "space", Action: "toggle-stage", Description: "stage or unstage selected path"},
	{Key: "a", Action: "stage-all", Description: "stage all tracked, untracked, and deleted paths"},
	{Key: "U", Action: "unstage-all", Description: "unstage all while preserving working-tree content"},
	{Key: "enter/d", Action: "open-diff", Description: "open selected file diff"},
	{Key: "/", Action: "filter", Description: "filter files"},
	{Key: "S", Action: "sort", Description: "cycle status-file sort mode"},
	{Key: "!", Action: "conflicts", Description: "toggle conflict-only status filter"},
	{Key: "R", Action: "restore", Description: "guarded restore of selected tracked path", Destructive: true},
	{Key: "r", Action: "refresh", Description: "refresh repository"},
	{Key: "1", Action: "status", Description: "open status workspace"},
	{Key: "b", Action: "branches", Description: "open branches workspace"},
	{Key: "s", Action: "stashes", Description: "open stashes workspace"},
	{Key: "l", Action: "history", Description: "open history workspace"},
	{Key: "n", Action: "remotes", Description: "open remotes workspace"},
	{Key: "w", Action: "worktrees", Description: "open worktrees workspace"},
	{Key: "v", Action: "repositories", Description: "open repositories workspace"},
	{Key: "c", Action: "commit", Description: "open commit workspace"},
	{Key: "ctrl+p", Action: "palette", Description: "open command palette"},
	{Key: "?", Action: "help", Description: "open full help"},
	{Key: "esc", Action: "close", Description: "close overlay or pane"},
	{Key: "q", Action: "quit", Description: "quit gitwatch"},
}

func HelpLines() []string {
	out := make([]string, len(DefaultBindings))
	for i, b := range DefaultBindings {
		out[i] = b.Key + "  " + b.Description
	}
	return out
}
