package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/sphireinc/git-watch/internal/sequencer"
)

type ConflictChoice string

const (
	ChooseOurs        ConflictChoice = "ours"
	ChooseTheirs      ConflictChoice = "theirs"
	ChooseBoth        ConflictChoice = "both"
	MarkResolved      ConflictChoice = "resolved"
	RestoreUnresolved ConflictChoice = "unresolved"
)

// ResolveConflict performs one explicit whole-file conflict action. It does
// not stage choose-ours/theirs content; staging is a separate deliberate
// action so the authoritative index remains the source of resolution state.
func (r Runner) ResolveConflict(ctx context.Context, path []byte, choice ConflictChoice) (OperationResult, error) {
	if len(path) == 0 {
		return OperationResult{Name: "resolve conflict"}, fmt.Errorf("conflict path is required")
	}
	var args []string
	name := "conflict " + string(choice)
	switch choice {
	case ChooseOurs:
		args = []string{"checkout", "--ours", "--"}
	case ChooseTheirs:
		args = []string{"checkout", "--theirs", "--"}
	case ChooseBoth:
		// Retain both sides' merge presentation; the user must review/edit and
		// explicitly mark the result resolved.
		args = []string{"checkout", "--conflict=merge", "--"}
	case MarkResolved:
		args = []string{"add", "--"}
	case RestoreUnresolved:
		args = []string{"checkout", "--conflict=merge", "--"}
	default:
		return OperationResult{Name: name}, fmt.Errorf("unsupported conflict action %q", choice)
	}
	result, err := r.Run(ctx, append(args, string(path))...)
	return OperationResult{Name: name, Paths: [][]byte{append([]byte(nil), path...)}, Result: result}, err
}

// OperationLifecycle executes only the lifecycle verbs supported by the
// observed Git operation. It never guesses a command from stderr or UI text.
func (r Runner) OperationLifecycle(ctx context.Context, kind sequencer.Kind, action string) (OperationResult, error) {
	command := kind.String()
	if command == "unknown" || kind == sequencer.KindBisect {
		return OperationResult{Name: action + " " + command}, fmt.Errorf("operation %s does not support %s", command, action)
	}
	switch action {
	case "continue", "abort":
	case "skip":
		if kind != sequencer.KindCherryPick && kind != sequencer.KindRevert {
			return OperationResult{Name: action + " " + command}, fmt.Errorf("operation %s does not support skip", command)
		}
	default:
		return OperationResult{Name: action + " " + command}, fmt.Errorf("unsupported operation lifecycle action %q", action)
	}
	result, err := r.Run(ctx, command, "--"+action)
	return OperationResult{Name: action + " " + command, Result: result}, err
}

// ExternalMergeToolCommand returns a typed Git command for the user's
// configured mergetool. The command is only started after an explicit UI
// action; returning it does not mutate repository state.
func (r Runner) ExternalMergeToolCommand(path []byte) (*exec.Cmd, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("conflict path is required")
	}
	binary := r.Binary
	if binary == "" {
		binary = "git"
	}
	command := exec.Command(binary, "mergetool", "--no-prompt", "--", string(path))
	command.Dir = r.Dir
	if len(r.Env) > 0 {
		command.Env = append(os.Environ(), r.Env...)
	}
	return command, nil
}
