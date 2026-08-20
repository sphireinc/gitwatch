package git

import (
	"context"
	"errors"
	"fmt"

	"github.com/jusanchez/gitwatch/internal/repo"
)

type OperationResult struct {
	Name   string
	Paths  [][]byte
	Result Result
}

func (r Runner) Stage(ctx context.Context, path []byte) (OperationResult, error) {
	return r.pathOperation(ctx, "stage", []byte(path), "add", "--")
}

func StageAllowed(entry repo.Entry) bool { return !entry.Conflicted }

func (r Runner) Unstage(ctx context.Context, path []byte) (OperationResult, error) {
	result, err := r.pathOperation(ctx, "unstage", path, "restore", "--staged", "--")
	if err == nil {
		return result, nil
	}
	if !errors.Is(err, ErrCommandFailed) {
		return result, err
	}
	return r.pathOperation(ctx, "unstage", path, "reset", "HEAD", "--")
}

func (r Runner) StageAll(ctx context.Context) (OperationResult, error) {
	result, err := r.Run(ctx, "add", "--all", "--")
	return OperationResult{Name: "stage all", Result: result}, err
}

func (r Runner) UnstageAll(ctx context.Context) (OperationResult, error) {
	result, err := r.Run(ctx, "restore", "--staged", ".")
	if err != nil {
		result, err = r.Run(ctx, "reset", "HEAD", "--", ".")
	}
	return OperationResult{Name: "unstage all", Result: result}, err
}

func (r Runner) pathOperation(ctx context.Context, name string, path []byte, args ...string) (OperationResult, error) {
	argv := append([]string{}, args...)
	argv = append(argv, string(path))
	result, err := r.Run(ctx, argv...)
	if err != nil {
		return OperationResult{Name: name, Paths: [][]byte{append([]byte(nil), path...)}, Result: result}, fmt.Errorf("%s %q: %w", name, path, err)
	}
	return OperationResult{Name: name, Paths: [][]byte{append([]byte(nil), path...)}, Result: result}, nil
}
