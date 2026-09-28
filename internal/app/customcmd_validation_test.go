package app

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/customcmd"
	"github.com/sphireinc/git-watch/internal/git"
)

func TestCustomCommandSelectionValidationBlocksLaunchAndShowsError(t *testing.T) {
	m := New()
	defer func() { _ = m.Close() }()
	m.Discovery = git.Discovery{Root: t.TempDir()}
	m.CustomCommands = []customcmd.Definition{{
		Name: "choose", Executable: "tool", Args: []string{"{prompt:choice}"},
		Prompts: []customcmd.Prompt{{ID: "choice", Label: "Choice", Kind: customcmd.PromptSelect, Options: []string{"invalid", "valid"}, Pattern: "^valid$"}},
	}}
	if cmd := m.runCustomCommand("choose"); cmd != nil || m.CustomCommandForm == nil {
		t.Fatal("command launched before form submit")
	}
	if cmd := m.updateCustomCommandForm("enter"); cmd != nil || m.CustomCommandForm == nil || !strings.Contains(m.Status, "invalid value") {
		t.Fatalf("invalid choice: cmd=%v form=%v status=%q", cmd != nil, m.CustomCommandForm != nil, m.Status)
	}
	if cmd := m.updateCustomCommandForm("down"); cmd != nil {
		t.Fatal("command launched during correction")
	}
	if cmd := m.updateCustomCommandForm("enter"); cmd == nil || m.CustomCommandForm != nil {
		t.Fatal("valid correction did not produce a command")
	}
}
