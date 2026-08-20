package app

type Binding struct {
	Key, Action, Description string
	Destructive              bool
}

var DefaultBindings = []Binding{
	{Key: "↑/↓, j/k", Action: "move", Description: "move selection"},
	{Key: "space", Action: "toggle-stage", Description: "stage or unstage selected path"},
	{Key: "a", Action: "stage-all", Description: "stage all repository changes"},
	{Key: "U", Action: "unstage-all", Description: "unstage all repository changes"},
	{Key: "enter/d", Action: "open-diff", Description: "open selected file diff"},
	{Key: "/", Action: "filter", Description: "filter files"},
	{Key: "s", Action: "sort", Description: "cycle sort mode"},
	{Key: "r", Action: "refresh", Description: "refresh repository"},
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
