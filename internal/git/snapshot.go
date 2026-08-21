package git

import (
	"context"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
)

func Snapshot(ctx context.Context, d Discovery, generation uint64) (repo.Snapshot, error) {
	start := time.Now()
	result, err := NewRunner(d.Root).Run(ctx, "status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all")
	if err != nil {
		return repo.Snapshot{}, err
	}
	status, err := ParseStatus(result.Stdout)
	if err != nil {
		return repo.Snapshot{}, err
	}
	s := repo.Snapshot{Root: d.Root, GitDir: d.GitDir, Generation: generation, ObservedAt: time.Now(), RefreshDuration: time.Since(start), Branch: repo.Branch{Name: status.BranchHead, OID: status.BranchOID, Upstream: status.Upstream, Ahead: status.Ahead, Behind: status.Behind, Detached: d.Detached, Unborn: d.Unborn}}
	for _, raw := range status.Entries {
		e := repo.Entry{Path: repo.Path(raw.Path), Original: repo.Path(raw.OrigPath), Kind: raw.Kind, XY: raw.XY, Renamed: raw.Kind == '2' && len(raw.RenameScore) > 0 && raw.RenameScore[0] == 'R', Copied: raw.Kind == '2' && len(raw.RenameScore) > 0 && raw.RenameScore[0] == 'C', Untracked: raw.Kind == '?', Conflicted: raw.Kind == 'u', ModeHead: raw.ModeHead, ModeIndex: raw.ModeIndex, ModeWork: raw.ModeWork}
		if raw.Kind == '1' || raw.Kind == '2' || raw.Kind == 'u' {
			e.Staged = len(raw.XY) > 0 && raw.XY[0] != '.'
			e.Unstaged = len(raw.XY) > 1 && raw.XY[1] != '.'
		}
		e.Deleted = len(raw.XY) == 2 && (raw.XY[0] == 'D' || raw.XY[1] == 'D')
		s.Entries = append(s.Entries, e)
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
	return s, nil
}
