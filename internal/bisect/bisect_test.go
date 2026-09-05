package bisect

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestStartLoadMarkAndResetBisect(t *testing.T) {
	ctx := context.Background()
	dir, runner := bisectRepository(t)
	commit := func(content, message string) string {
		if err := os.WriteFile(filepath.Join(dir, "file"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("file")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout))
	}
	good := commit("good\n", "good")
	bad := commit("bad\n", "bad")
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	outcome := Start(ctx, runner, StartRequest{Repository: discovery.Root, Generation: 4, Bad: bad, Good: good})
	if outcome.Err != nil || !outcome.State.Active {
		t.Fatalf("start outcome = %#v", outcome)
	}
	loaded, err := Load(ctx, git.NewRunner(dir), discovery.Root, 5)
	if err != nil || !loaded.Active || loaded.Bad != bad || loaded.Good != good || loaded.Candidate == "" || len(loaded.Log) == 0 {
		t.Fatalf("loaded state = %#v, err=%v", loaded, err)
	}
	marked := MarkCandidate(ctx, git.NewRunner(dir), Request{Repository: discovery.Root, Generation: 6, Mark: Bad})
	if marked.Err != nil {
		t.Fatalf("mark bad outcome = %#v", marked)
	}
	reset := Reset(ctx, git.NewRunner(dir), Request{Repository: discovery.Root, Generation: 7})
	if reset.Err != nil || reset.State.Active {
		t.Fatalf("reset outcome = %#v", reset)
	}
}

func TestBisectSkipAndRestartFromFreshRunner(t *testing.T) {
	ctx := context.Background()
	dir, runner := bisectRepository(t)
	commit := func(content, message string) string {
		if err := os.WriteFile(filepath.Join(dir, "file"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte("file")); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Commit(ctx, git.CommitOptions{Message: []byte(message + "\n")}); err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(mustRun(t, runner, ctx, "rev-parse", "HEAD").Stdout))
	}
	good := commit("one\n", "one")
	commit("two\n", "two")
	commit("three\n", "three")
	bad := commit("four\n", "four")
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if outcome := Start(ctx, runner, StartRequest{Repository: discovery.Root, Generation: 1, Bad: bad, Good: good}); outcome.Err != nil {
		t.Fatalf("start outcome = %#v", outcome)
	}
	if outcome := MarkCandidate(ctx, git.NewRunner(dir), Request{Repository: discovery.Root, Generation: 2, Mark: Skip}); outcome.Err != nil {
		t.Fatalf("skip outcome = %#v", outcome)
	}
	state, err := Load(ctx, git.NewRunner(dir), discovery.Root, 3)
	if err != nil || !state.Active {
		t.Fatalf("restarted state = %#v, err=%v", state, err)
	}
	if len(state.Log) < 2 {
		t.Fatalf("bisect log after restart = %#v", state.Log)
	}
	if outcome := Reset(ctx, git.NewRunner(dir), Request{Repository: discovery.Root, Generation: 4}); outcome.Err != nil {
		t.Fatalf("reset outcome = %#v", outcome)
	}
}

func TestBisectRejectsDirtyStartAndInvalidRefs(t *testing.T) {
	ctx := context.Background()
	dir, runner := bisectRepository(t)
	discovery, err := git.Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if outcome := Start(ctx, runner, StartRequest{Repository: discovery.Root, Bad: "--bad", Good: "good"}); outcome.Err != ErrInvalidRef {
		t.Fatalf("invalid ref error = %v", outcome.Err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dirty"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if outcome := Start(ctx, runner, StartRequest{Repository: discovery.Root, Bad: "HEAD", Good: "HEAD~1"}); outcome.Err != ErrDirtyWorktree {
		t.Fatalf("dirty start error = %v", outcome.Err)
	}
}

func bisectRepository(t *testing.T) (string, git.Runner) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	runner := git.NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	return dir, runner
}

func mustRun(t *testing.T, runner git.Runner, ctx context.Context, args ...string) git.Result {
	t.Helper()
	result, err := runner.Run(ctx, args...)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
