package git

import (
	"context"
	"fmt"
)

type ConflictChoice string

const (
	ChooseOurs        ConflictChoice = "ours"
	ChooseTheirs      ConflictChoice = "theirs"
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
