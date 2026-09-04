package cherrypick

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestExecuteOrderedCommitsAndCaptureJournal(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := setupRepository(t, dir)
	write := func(value string) {
		if err := os.WriteFile(filepath.Join(dir, "file"), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("file")); err != nil {
			t.Fatal(err)
		}
	}
	write("base\n")
	first, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("first\n")})
	if err != nil {
		t.Fatal(err)
	}
	write("base\nfirst\n")
	second, err := runner.Commit(ctx, git.CommitOptions{Message: []byte("second\n")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "-c", "target"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "reset", "--hard", "HEAD~2"); err != nil {
		t.Fatal(err)
	}
	engine := Engine{Runner: runner, Repository: dir, Generation: 7}
	outcome := engine.Execute(ctx, Request{Repository: dir, Generation: 7, SHAs: []string{first.SHA, second.SHA}})
	if outcome.Err != nil || outcome.Paused {
		t.Fatalf("cherry-pick outcome = %#v", outcome)
	}
	if outcome.Journal.OriginalHEAD == "" || len(outcome.Journal.SHAs) != 2 {
		t.Fatalf("journal = %#v", outcome.Journal)
	}
	show, err := runner.Run(ctx, "show", "-s", "--format=%s", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(show.Stdout)); got != "second" {
		t.Fatalf("HEAD subject = %q", got)
	}
}

func TestMergeRequiresExplicitMainline(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := setupRepository(t, dir)
	writeCommit := func(name, value string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte(name)); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(name + "\n")}); err != nil {
			t.Fatal(err)
		}
	}
	writeCommit("base", "base")
	if _, err := runner.Run(ctx, "switch", "-c", "side"); err != nil {
		t.Fatal(err)
	}
	writeCommit("side", "side")
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	writeCommit("main", "main")
	if _, err := runner.Run(ctx, "merge", "--no-ff", "side", "-m", "merge"); err != nil {
		t.Fatal(err)
	}
	result, err := runner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	engine := Engine{Runner: runner, Repository: dir}
	if outcome := engine.Execute(ctx, Request{Repository: dir, SHAs: []string{strings.TrimSpace(string(result.Stdout))}}); outcome.Err == nil || !strings.Contains(outcome.Err.Error(), "mainline") {
		t.Fatalf("missing mainline outcome = %#v", outcome)
	}
}

func setupRepository(t *testing.T, dir string) git.Runner {
	t.Helper()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "seed"), []byte("seed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(context.Background(), []byte("seed")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(context.Background(), git.CommitOptions{Message: []byte("seed\n")}); err != nil {
		t.Fatal(err)
	}
	return runner
}
