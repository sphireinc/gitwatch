package branches

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestParse(t *testing.T) {
	v := Parse([]byte("main\x00abc\x00origin/main\x00*\x00>\x001700000000\x00initial commit\nfeature\x00def\x00\x00 \x00=\x001600000000\x00feature commit\n"))
	if len(v) != 2 || !v[0].Current || v[0].Ahead != 1 || v[0].LastCommitUnix != 1700000000 || v[0].Subject != "initial commit" || v[1].Remote {
		t.Fatal(v)
	}
}

func TestParseIdentifiesQualifiedRemoteRef(t *testing.T) {
	entries := Parse([]byte("origin/main\x00abc\x00\x00 \x00=\x001700000000\x00remote commit\x00refs/remotes/origin/main\n"))
	if len(entries) != 1 || !entries[0].Remote || entries[0].RemoteName != "origin" || entries[0].RemoteBranch != "main" {
		t.Fatalf("remote entry = %#v", entries)
	}
}

func TestDeleteGuardsCurrentBranchAndExactConfirmation(t *testing.T) {
	branch := Branch{Name: "feature", Current: false}
	if _, err := Delete(context.Background(), git.Runner{}, branch, DeletePrompt("feature", false), "wrong"); !errors.Is(err, ErrConfirmation) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	branch.Current = true
	if _, err := Delete(context.Background(), git.Runner{}, branch, DeletePrompt("feature", false), "feature"); !errors.Is(err, ErrCurrentBranch) {
		t.Fatalf("expected current branch error, got %v", err)
	}
}

func TestListWithOccupancy(t *testing.T) {
	// The helper preserves branch parsing while attaching only local worktree occupancy.
	entries := []Branch{{Name: "main"}, {Name: "remotes/origin/main", Remote: true}}
	occupancy := map[string]string{"main": "/tmp/main"}
	AttachOccupancy(entries, occupancy)
	if entries[0].OccupiedPath != "/tmp/main" || entries[1].OccupiedPath != "" {
		t.Fatalf("occupancy = %#v", entries)
	}
}

func TestListUsesGitNulFormatAtom(t *testing.T) {
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	if _, err := runner.Run(context.Background(), "init", "-b", "main", "--", dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "add", "--", "file.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "-c", "user.name=gitwatch", "-c", "user.email=gitwatch@example.com", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	entries, err := List(context.Background(), runner)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "main" || !entries[0].Current {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestListKeepsSameBranchNameDistinctAcrossRemotes(t *testing.T) {
	localDir := t.TempDir()
	originDir := t.TempDir()
	backupDir := t.TempDir()
	local := git.NewRunner(localDir)
	origin := git.NewRunner(originDir)
	backup := git.NewRunner(backupDir)
	runBranchTestGit(t, origin, "init", "--bare")
	runBranchTestGit(t, backup, "init", "--bare")
	runBranchTestGit(t, local, "init", "-b", "main")
	runBranchTestGit(t, local, "config", "user.name", "gitwatch test")
	runBranchTestGit(t, local, "config", "user.email", "gitwatch@example.test")
	if err := os.WriteFile(filepath.Join(localDir, "file"), []byte("content\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runBranchTestGit(t, local, "add", "file")
	runBranchTestGit(t, local, "-c", "commit.gpgsign=false", "commit", "-m", "initial")
	runBranchTestGit(t, local, "remote", "add", "origin", originDir)
	runBranchTestGit(t, local, "remote", "add", "backup", backupDir)
	runBranchTestGit(t, local, "push", "--set-upstream", "origin", "main")
	runBranchTestGit(t, local, "push", "backup", "main")
	runBranchTestGit(t, local, "fetch", "backup")

	entries, err := List(context.Background(), local)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]Branch{}
	for _, entry := range entries {
		if entry.Remote && (entry.Name == "origin/main" || entry.Name == "backup/main") {
			seen[entry.Name] = entry
		}
	}
	if len(seen) != 2 || seen["origin/main"].RemoteName != "origin" || seen["backup/main"].RemoteName != "backup" {
		t.Fatalf("remote branch rows = %#v", seen)
	}
	for name, entry := range seen {
		if entry.Ahead != 0 || entry.Behind != 0 {
			t.Fatalf("%s divergence = ahead %d behind %d", name, entry.Ahead, entry.Behind)
		}
	}
}

func runBranchTestGit(t *testing.T, runner git.Runner, args ...string) {
	t.Helper()
	if _, err := runner.Run(context.Background(), args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}
