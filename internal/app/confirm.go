package app

type Confirmation struct {
	Open                        bool
	Action, Path, Scope, Prompt string
}

func RestoreConfirmation(path string, staged, unstaged bool) Confirmation {
	scope := "working-tree content"
	if staged && unstaged {
		scope = "staged and working-tree content"
	} else if staged {
		scope = "staged content"
	}
	return Confirmation{Open: true, Action: "restore", Path: path, Scope: scope, Prompt: "Restore " + path + " and discard " + scope + "?"}
}

func (c Confirmation) Accept(input string) bool { return c.Open && input == "yes" }
