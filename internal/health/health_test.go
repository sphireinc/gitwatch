package health

import (
	"testing"
	"time"

	"github.com/sphireinc/git-watch/internal/repo"
)

func TestComputeUsesAuthoritativeLocalStateAndSemanticSeverity(t *testing.T) {
	observed := time.Now()
	snapshot := repo.Snapshot{ObservedAt: observed, Branch: repo.Branch{Ahead: 2, Behind: 1}, Counts: repo.Counts{Unstaged: 3, Conflicted: 1}}
	summary := Compute(snapshot, 4, 2, []string{"remote summary unavailable"})
	if summary.Severity != SeverityCritical || summary.Dirty != 3 || summary.Unpushed != 2 || summary.Stashes != 4 || summary.Worktrees != 2 || !summary.FreshAt.Equal(observed) {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.Attention) != 2 || summary.Attention[0] != "conflicts" {
		t.Fatalf("attention = %#v", summary.Attention)
	}
}

func TestComputeOfflineCleanRepositoryRemainsHealthy(t *testing.T) {
	summary := Compute(repo.Snapshot{ObservedAt: time.Now()}, 0, 0, nil)
	if summary.Severity != SeverityHealthy || summary.Source != "git status" {
		t.Fatalf("summary = %#v", summary)
	}
}
