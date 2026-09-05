// Package undo contains conservative, operation-specific recovery policies.
package undo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/repo"
)

var (
	ErrUnsupportedOperation = errors.New("operation has no safe automatic undo policy")
	ErrRepositoryMismatch   = errors.New("undo record belongs to another repository")
	ErrDiverged             = errors.New("repository diverged from the recorded post-operation state")
	ErrActiveOperation      = errors.New("repository has an active Git operation")
	ErrInvalidHead          = errors.New("undo record contains an invalid object ID")
)

// Request is the evidence required to undo one operation safely.
type Request struct {
	Repository       string
	Kind             string
	Ref              string
	OldHead          string
	NewHead          string
	PostSnapshotHash string
	Discovery        git.Discovery
	Generation       uint64
}

// Outcome contains the authoritative state observed after the attempted undo.
type Outcome struct {
	Result   git.Result
	Snapshot repo.Snapshot
	Err      error
}

// Plan validates a commit undo without invoking Git. It intentionally exposes
// only reset --soft for this first policy; no generic reset or hard reset is
// ever synthesized.
func Plan(request Request, snapshot repo.Snapshot) ([]string, error) {
	if request.Repository == "" || request.Repository != snapshot.Root {
		return nil, ErrRepositoryMismatch
	}
	if request.Kind != "commit" {
		return nil, ErrUnsupportedOperation
	}
	if snapshot.Operation != nil {
		return nil, ErrActiveOperation
	}
	if !validObjectID(request.OldHead) || !validObjectID(request.NewHead) || request.OldHead == request.NewHead {
		return nil, ErrInvalidHead
	}
	if snapshot.Branch.OID != request.NewHead || request.Ref == "" || snapshot.Branch.Name != request.Ref {
		return nil, ErrDiverged
	}
	if request.PostSnapshotHash == "" || SnapshotFingerprint(snapshot) != request.PostSnapshotHash {
		return nil, ErrDiverged
	}
	return []string{"reset", "--soft", request.OldHead}, nil
}

// Execute re-reads authoritative status immediately before the guarded
// mutation, then refreshes it again after Git completes.
func Execute(ctx context.Context, runner git.Runner, request Request) Outcome {
	discovery := request.Discovery
	if discovery.Root == "" {
		return Outcome{Err: ErrRepositoryMismatch}
	}
	snapshot, err := git.Snapshot(ctx, discovery, request.Generation)
	if err != nil {
		return Outcome{Err: fmt.Errorf("undo status: %w", err)}
	}
	args, err := Plan(request, snapshot)
	if err != nil {
		return Outcome{Snapshot: snapshot, Err: err}
	}
	result, err := runner.Run(ctx, args...)
	outcome := Outcome{Result: result, Snapshot: snapshot, Err: err}
	if err == nil {
		if refreshed, refreshErr := git.Snapshot(ctx, discovery, request.Generation); refreshErr == nil {
			outcome.Snapshot = refreshed
		} else {
			outcome.Err = fmt.Errorf("undo refresh: %w", refreshErr)
		}
	}
	return outcome
}

// SnapshotFingerprint deterministically represents index/worktree state while
// excluding HEAD, so a commit undo can preserve content without accepting
// unrelated edits made after the recorded operation.
func SnapshotFingerprint(snapshot repo.Snapshot) string {
	type item struct{ key string }
	items := make([]item, 0, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		key := fmt.Sprintf("%d:%s:%s:%s:%s:%s:%t:%t:%t:%t:%t:%t", len(entry.Path), entry.Path, entry.Original, entry.XY, entry.ModeHead, entry.ModeIndex, entry.Staged, entry.Unstaged, entry.Untracked, entry.Conflicted, entry.Deleted, entry.Renamed)
		items = append(items, item{key: key})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].key < items[j].key })
	hash := sha256.New()
	for _, item := range items {
		_, _ = hash.Write([]byte(item.key))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validObjectID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	return strings.Trim(value, "0123456789abcdefABCDEF") == ""
}
