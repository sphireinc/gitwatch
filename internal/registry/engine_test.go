package registry

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

func TestEngineUsesBoundedWorkersAndCachesInactiveRepositories(t *testing.T) {
	var active atomic.Int32
	var peak atomic.Int32
	var calls atomic.Int32
	engine := NewEngine(2)
	engine.InactiveAfter = time.Hour
	engine.Discover = func(context.Context, string) (git.Discovery, error) {
		current := active.Add(1)
		for {
			old := peak.Load()
			if current <= old || peak.CompareAndSwap(old, current) {
				break
			}
		}
		defer active.Add(-1)
		calls.Add(1)
		return git.Discovery{Root: "repo"}, nil
	}
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = func(context.Context, git.Discovery) (int, error) { return 3, nil }
	engine.Remotes = func(context.Context, git.Discovery) (int, error) { return 2, nil }
	entries := []Repository{{Path: "one"}, {Path: "two"}, {Path: "three"}}
	if got := engine.Refresh(context.Background(), entries, "one"); len(got) != 3 || peak.Load() > 2 || got[0].Stashes != 3 || got[0].Remotes != 2 {
		t.Fatalf("unexpected refresh: len=%d peak=%d", len(got), peak.Load())
	}
	if calls.Load() != 3 {
		t.Fatalf("unexpected discovery count: %d", calls.Load())
	}
	entries[1].LastOpened = time.Now().Add(-2 * time.Hour)
	entries[2].LastOpened = time.Now().Add(-2 * time.Hour)
	engine.Refresh(context.Background(), entries, "one")
	if calls.Load() != 4 {
		t.Fatalf("inactive cache was not used: %d", calls.Load())
	}
}

func TestInspectGitignoreHealthReportsManagedAndMissingStates(t *testing.T) {
	root := t.TempDir()
	if health := inspectGitignore(root); health.Exists || health.Managed != 0 {
		t.Fatalf("missing health=%+v", health)
	}
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	template, _ := cat.Get("root/Go")
	content, err := managed.EncodeManagedBlock(template.ID, "github/gitignore", cat.Version(), template.ContentSHA256, template.Content, []byte("\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), content, 0644); err != nil {
		t.Fatal(err)
	}
	health := inspectGitignore(root)
	if !health.Exists || health.Managed != 1 || health.Attention != 0 {
		t.Fatalf("managed health=%+v", health)
	}
}

func TestEngineUsesRepositoryRefreshPolicy(t *testing.T) {
	engine := NewEngine(1)
	engine.InactiveAfter = time.Hour
	engine.InactiveAfterFor = func(repository Repository) time.Duration {
		if len(repository.Groups) > 0 && repository.Groups[0] == "fast" {
			return time.Minute
		}
		return time.Hour
	}
	engine.Discover = func(context.Context, string) (git.Discovery, error) { return git.Discovery{}, nil }
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = nil
	engine.Remotes = nil
	entries := []Repository{{Path: "fast", Groups: []string{"fast"}, LastOpened: time.Now().Add(-2 * time.Hour)}, {Path: "slow", LastOpened: time.Now().Add(-30 * time.Minute)}}
	engine.Refresh(context.Background(), entries, "active")
	results := engine.Refresh(context.Background(), entries, "active")
	if !results[0].Skipped || results[1].Skipped {
		t.Fatalf("refresh policy results = %#v", results)
	}
}

func TestEngineBudgetCancelsSlowStatusSource(t *testing.T) {
	engine := NewEngine(1)
	engine.Budget = 10 * time.Millisecond
	engine.Stashes, engine.Remotes = nil, nil
	engine.Discover = func(ctx context.Context, _ string) (git.Discovery, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return git.Discovery{}, nil
		case <-ctx.Done():
			return git.Discovery{}, ctx.Err()
		}
	}
	result := engine.Refresh(context.Background(), []Repository{{Path: "slow"}}, "active")
	if len(result) != 1 || !errors.Is(result[0].Error, context.DeadlineExceeded) {
		t.Fatalf("slow source result = %#v", result)
	}
}

func TestEngineRecordsAuxiliaryWarningsAndRefreshMetadata(t *testing.T) {
	engine := NewEngine(1)
	engine.Discover = func(context.Context, string) (git.Discovery, error) { return git.Discovery{Root: "/repo"}, nil }
	engine.Snapshot = func(context.Context, git.Discovery, uint64) (repo.Snapshot, error) { return repo.Snapshot{}, nil }
	engine.Stashes = func(context.Context, git.Discovery) (int, error) { return 0, errors.New("stash unavailable") }
	engine.Remotes = func(context.Context, git.Discovery) (int, error) { return 0, errors.New("remote unavailable") }
	results := engine.Refresh(context.Background(), []Repository{{Path: "/repo", Name: "repo"}}, "/repo")
	if len(results) != 1 || len(results[0].Warnings) != 2 || results[0].Refreshed.IsZero() || results[0].Duration < 0 {
		t.Fatalf("refresh metadata = %#v", results)
	}
}

func TestEngineKeepsMixedAdvancedAttentionAcrossTwentyRepositories(t *testing.T) {
	engine := NewEngine(4)
	engine.Stashes, engine.Remotes = nil, nil
	root := t.TempDir()
	repositories := make([]Repository, 20)
	for i := range repositories {
		path := root + fmt.Sprintf("/repo-%02d", i)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		repositories[i] = Repository{Path: path, Name: fmt.Sprintf("repo-%02d", i)}
	}
	engine.Discover = func(_ context.Context, path string) (git.Discovery, error) {
		if path == repositories[7].Path {
			return git.Discovery{}, errors.New("repository disappeared")
		}
		return git.Discovery{Root: path}, nil
	}
	engine.Snapshot = func(_ context.Context, discovery git.Discovery, _ uint64) (repo.Snapshot, error) {
		snapshot := repo.Snapshot{Root: discovery.Root, Branch: repo.Branch{Name: "main"}}
		if discovery.Root == repositories[3].Path {
			operation, err := sequencer.NewState(sequencer.RepositoryID(discovery.Root), 0, sequencer.KindRebase, sequencer.PhaseActive)
			if err != nil {
				return repo.Snapshot{}, err
			}
			snapshot.Operation = &operation
		}
		if discovery.Root == repositories[5].Path {
			snapshot.Counts.Conflicted = 1
		}
		if discovery.Root == repositories[9].Path {
			snapshot.Counts.Untracked = 1
		}
		return snapshot, nil
	}
	results := engine.Refresh(context.Background(), repositories, repositories[0].Path)
	rows := Rows(results)
	if len(rows) != len(repositories) {
		t.Fatalf("mixed attention row count = %d", len(rows))
	}
	byPath := make(map[string]Row, len(rows))
	for _, row := range rows {
		byPath[row.Repository.Path] = row
	}
	if byPath[repositories[3].Path].Attention != "rebase" || byPath[repositories[5].Path].Attention != "conflict" || byPath[repositories[9].Path].Attention != "dirty/diverged" {
		t.Fatalf("mixed attention rows = %#v", rows)
	}
	if byPath[repositories[7].Path].State != "error" || byPath[repositories[0].Path].State == "error" {
		t.Fatalf("broken repository isolation = %#v", rows)
	}
}
