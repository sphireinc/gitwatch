// Package bisect provides typed, repository-scoped Git bisect operations.
package bisect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/repo"
)

var (
	ErrInvalidRef      = errors.New("bisect ref is invalid")
	ErrNotActive       = errors.New("bisect is not active")
	ErrAlreadyActive   = errors.New("bisect is already active")
	ErrDirtyWorktree   = errors.New("bisect requires a clean worktree")
	ErrActiveOperation = errors.New("another Git operation is active")
)

const maxLogBytes = 1 << 20

type Mark string

const (
	Good Mark = "good"
	Bad  Mark = "bad"
	Skip Mark = "skip"
)

// State is the bounded projection reconstructed from Git's bisect metadata.
type State struct {
	Repository string
	Generation uint64
	Active     bool
	Good       string
	Bad        string
	Candidate  string
	Log        []string
}

func (s State) Clone() State {
	s.Log = append([]string(nil), s.Log...)
	return s
}

type StartRequest struct {
	Repository string
	Generation uint64
	Bad        string
	Good       string
}

type Request struct {
	Repository string
	Generation uint64
	Mark       Mark
}

type Outcome struct {
	Result   git.Result
	Snapshot repo.Snapshot
	State    State
	Err      error
}

// Load reconstructs bisect state from Git and never infers activity from the
// UI or from a prior gitwatch process.
func Load(ctx context.Context, runner git.Runner, repository string, generation uint64) (State, error) {
	state := State{Repository: repository, Generation: generation}
	startPath, err := metadataPath(ctx, runner, "BISECT_START")
	if err != nil {
		return State{}, fmt.Errorf("bisect start path: %w", err)
	}
	if _, err := os.Stat(startPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return state, nil
		}
		return State{}, fmt.Errorf("bisect start state: %w", err)
	}
	if startPath == "" {
		return state, nil
	}
	state.Active = true
	state.Candidate, _, err = verify(ctx, runner, "HEAD")
	if err != nil {
		return State{}, fmt.Errorf("bisect candidate: %w", err)
	}
	result, err := runner.RunBounded(ctx, maxLogBytes, "bisect", "log")
	if err != nil {
		return State{}, fmt.Errorf("bisect log: %w", err)
	}
	state.Log = boundedLines(result.Stdout)
	state.Bad = boundary(state.Log, "# bad:")
	state.Good = boundary(state.Log, "# good:")
	return state, nil
}

// Start begins a bisect only with explicit, non-option refs and a clean,
// operation-free repository snapshot.
func Start(ctx context.Context, runner git.Runner, request StartRequest) Outcome {
	if request.Repository == "" || !validRef(request.Bad) || !validRef(request.Good) || request.Bad == request.Good {
		return Outcome{Err: ErrInvalidRef}
	}
	state, err := Load(ctx, runner, request.Repository, request.Generation)
	if err != nil {
		return Outcome{Err: err}
	}
	if state.Active {
		return Outcome{State: state, Err: ErrAlreadyActive}
	}
	snapshot, err := git.Snapshot(ctx, git.Discovery{Root: request.Repository}, request.Generation)
	if err != nil {
		return Outcome{Err: fmt.Errorf("bisect preflight: %w", err)}
	}
	if snapshot.Operation != nil {
		return Outcome{Snapshot: snapshot, Err: ErrActiveOperation}
	}
	if dirty(snapshot) {
		return Outcome{Snapshot: snapshot, Err: ErrDirtyWorktree}
	}
	result, err := runner.Run(ctx, "bisect", "start", request.Bad, request.Good)
	return finish(ctx, runner, request.Repository, request.Generation, result, err)
}

// Mark records one explicit bisect result for the current candidate.
func MarkCandidate(ctx context.Context, runner git.Runner, request Request) Outcome {
	if request.Repository == "" || !validMark(request.Mark) {
		return Outcome{Err: ErrInvalidRef}
	}
	state, err := Load(ctx, runner, request.Repository, request.Generation)
	if err != nil {
		return Outcome{Err: err}
	}
	if !state.Active {
		return Outcome{State: state, Err: ErrNotActive}
	}
	result, err := runner.Run(ctx, "bisect", string(request.Mark))
	return finish(ctx, runner, request.Repository, request.Generation, result, err)
}

func Reset(ctx context.Context, runner git.Runner, request Request) Outcome {
	if request.Repository == "" {
		return Outcome{Err: ErrInvalidRef}
	}
	state, err := Load(ctx, runner, request.Repository, request.Generation)
	if err != nil {
		return Outcome{Err: err}
	}
	if !state.Active {
		return Outcome{State: state, Err: ErrNotActive}
	}
	result, err := runner.Run(ctx, "bisect", "reset")
	return finish(ctx, runner, request.Repository, request.Generation, result, err)
}

func finish(ctx context.Context, runner git.Runner, repository string, generation uint64, result git.Result, commandErr error) Outcome {
	outcome := Outcome{Result: result, Err: commandErr}
	if snapshot, err := git.Snapshot(ctx, git.Discovery{Root: repository}, generation); err == nil {
		outcome.Snapshot = snapshot
	} else if outcome.Err == nil {
		outcome.Err = fmt.Errorf("bisect refresh: %w", err)
	}
	if state, err := Load(ctx, runner, repository, generation); err == nil {
		outcome.State = state
	} else if outcome.Err == nil {
		outcome.Err = err
	}
	return outcome
}

func verify(ctx context.Context, runner git.Runner, ref string) (string, bool, error) {
	result, err := runner.Run(ctx, "rev-parse", "--verify", "-q", ref)
	if err == nil {
		return strings.TrimSpace(string(result.Stdout)), true, nil
	}
	var commandErr *git.CommandError
	if errors.As(err, &commandErr) && commandErr.Result.ExitCode == 1 {
		return "", false, nil
	}
	return "", false, err
}

func metadataPath(ctx context.Context, runner git.Runner, name string) (string, error) {
	result, err := runner.Run(ctx, "rev-parse", "--git-path", name)
	if err != nil {
		return "", err
	}
	path := strings.TrimSpace(string(result.Stdout))
	if !filepath.IsAbs(path) {
		path = filepath.Join(runner.Dir, path)
	}
	return filepath.Clean(path), nil
}

func dirty(snapshot repo.Snapshot) bool {
	return snapshot.Counts.Staged > 0 || snapshot.Counts.Unstaged > 0 || snapshot.Counts.Untracked > 0 || snapshot.Counts.Conflicted > 0
}

func validRef(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-") && !strings.ContainsAny(value, "\r\n\x00")
}

func validMark(mark Mark) bool { return mark == Good || mark == Bad || mark == Skip }

func boundedLines(data []byte) []string {
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}
	return lines
}

func boundary(lines []string, prefix string) string {
	for _, line := range lines {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		start := strings.IndexByte(line, '[')
		end := strings.IndexByte(line, ']')
		if start >= 0 && end > start+1 {
			return line[start+1 : end]
		}
	}
	return ""
}
