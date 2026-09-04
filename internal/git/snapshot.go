package git

import (
	"context"
	"time"

	"github.com/sphireinc/git-watch/internal/conflicts"
	"github.com/sphireinc/git-watch/internal/repo"
)

// Snapshot reads authoritative porcelain-v2 status without modifying the index.
func Snapshot(ctx context.Context, d Discovery, generation uint64) (repo.Snapshot, error) {
	start := time.Now()
	// Prevent the read-only status refresh from opportunistically rewriting the
	// index stat cache. Such a write is harmless to Git but becomes a watcher
	// event and can otherwise create a status-refresh feedback loop.
	runner := NewRunner(d.Root)
	runner.Env = []string{"GIT_OPTIONAL_LOCKS=0"}
	result, err := runner.Run(ctx, "--no-optional-locks", "status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all")
	if err != nil {
		return repo.Snapshot{}, err
	}
	status, err := ParseStatus(result.Stdout)
	if err != nil {
		return repo.Snapshot{}, err
	}
	indexResult, err := runner.Run(ctx, "ls-files", "-u", "-z")
	if err != nil {
		return repo.Snapshot{}, err
	}
	indexConflicts, err := conflicts.ParseIndex(indexResult.Stdout)
	if err != nil {
		return repo.Snapshot{}, err
	}
	operation, err := DetectOperationState(ctx, d, generation)
	if err != nil {
		return repo.Snapshot{}, err
	}
	s := repo.Snapshot{Root: d.Root, GitDir: d.GitDir, Generation: generation, ObservedAt: time.Now(), RefreshDuration: time.Since(start), Branch: repo.Branch{Name: status.BranchHead, OID: status.BranchOID, Upstream: status.Upstream, Ahead: status.Ahead, Behind: status.Behind, Detached: d.Detached, Unborn: d.Unborn}}
	statusConflicts := make([]conflicts.Status, 0, len(status.Entries))
	if operation.Found {
		s.Operation = &operation.State
		s.OperationDiagnostics = append([]string(nil), operation.Diagnostics...)
	}
	for _, raw := range status.Entries {
		e := repo.Entry{Path: repo.Path(raw.Path), Original: repo.Path(raw.OrigPath), Kind: raw.Kind, XY: raw.XY, Renamed: raw.Kind == '2' && len(raw.RenameScore) > 0 && raw.RenameScore[0] == 'R', Copied: raw.Kind == '2' && len(raw.RenameScore) > 0 && raw.RenameScore[0] == 'C', Untracked: raw.Kind == '?', Conflicted: raw.Kind == 'u', ModeHead: raw.ModeHead, ModeIndex: raw.ModeIndex, ModeWork: raw.ModeWork, Submodule: raw.Submodule}
		if raw.Kind == '1' || raw.Kind == '2' || raw.Kind == 'u' {
			e.Staged = len(raw.XY) > 0 && raw.XY[0] != '.'
			e.Unstaged = len(raw.XY) > 1 && raw.XY[1] != '.'
		}
		e.Deleted = len(raw.XY) == 2 && (raw.XY[0] == 'D' || raw.XY[1] == 'D')
		s.Entries = append(s.Entries, e)
		if raw.Kind == 'u' {
			statusConflicts = append(statusConflicts, conflicts.Status{Path: raw.Path, XY: raw.XY, Worktree: raw.XY})
		}
		if e.Staged {
			s.Counts.Staged++
		}
		if e.Unstaged {
			s.Counts.Unstaged++
		}
		if e.Untracked {
			s.Counts.Untracked++
		}
		if e.Conflicted {
			s.Counts.Conflicted++
		}
		if e.Deleted {
			s.Counts.Deleted++
		}
	}
	s.Conflicts = conflicts.Correlate(indexConflicts, statusConflicts)
	return s, nil
}
