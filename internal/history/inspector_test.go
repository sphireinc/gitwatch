package history

import (
	"context"
	"errors"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestParseStats(t *testing.T) {
	stats := parseStats([]byte("3\t1\tinternal/app/app.go\n-\t-\tassets/logo.bin\n"))
	if len(stats) != 2 || stats[0].Added != 3 || stats[0].Deleted != 1 || !stats[1].Binary {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if got := statPaths([]byte("3\t1\tfile.txt\n")); len(got) != 1 || got[0] != "file.txt" {
		t.Fatalf("unexpected paths: %#v", got)
	}
}

func TestParseStatsZPreservesUnusualPathBytes(t *testing.T) {
	stats := parseStats([]byte("2\t1\tpath with\t tab.txt\x00-\t-\tline\nname.bin\x00"))
	if len(stats) != 2 || stats[0].Path != "path with\t tab.txt" || !stats[1].Binary || stats[1].Path != "line\nname.bin" {
		t.Fatalf("unexpected NUL stats: %#v", stats)
	}
}

func TestResolveRefRejectsOptionLikeInput(t *testing.T) {
	if _, err := ResolveRef(context.Background(), git.Runner{}, "-bad"); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("expected invalid ref, got %v", err)
	}
}

func TestLoadCommitMetadataFromRealRepository(t *testing.T) {
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	ctx := context.Background()
	if _, err := runner.Run(ctx, "init", "-b", "main", "--", dir); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "-c", "user.name=gitwatch", "-c", "user.email=gitwatch@example.com", "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	commit, err := LoadCommit(ctx, runner, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if commit.SHA == "" || commit.Short == "" || commit.Author != "gitwatch" || commit.Subject != "initial" {
		t.Fatalf("commit = %#v", commit)
	}
}
