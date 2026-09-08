package remotes

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

type captureRemoteRunner struct {
	args    [][]string
	results []git.Result
}

func (r *captureRemoteRunner) Run(_ context.Context, args ...string) (git.Result, error) {
	r.args = append(r.args, append([]string(nil), args...))
	if len(r.results) > 0 {
		result := r.results[0]
		r.results = r.results[1:]
		return result, nil
	}
	return git.Result{Args: append([]string(nil), args...)}, nil
}

func TestPullRequiresExplicitStrategy(t *testing.T) {
	_, err := Pull(context.Background(), git.Runner{}, "origin", "main", "")
	if !errors.Is(err, ErrStrategyRequired) {
		t.Fatalf("expected explicit strategy error, got %v", err)
	}
}

func TestRemoteLifecycleBuildsTypedArgv(t *testing.T) {
	var runner captureRemoteRunner
	if _, err := Add(context.Background(), &runner, "origin", "https://example.com/repo.git"); err != nil {
		t.Fatal(err)
	}
	if _, err := Rename(context.Background(), &runner, "origin", "upstream"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetURL(context.Background(), &runner, "upstream", "https://example.com/other.git"); err != nil {
		t.Fatal(err)
	}
	if _, err := Prune(context.Background(), &runner, "upstream", true); err != nil {
		t.Fatal(err)
	}
	if _, err := Remove(context.Background(), &runner, "upstream"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"remote", "add", "origin", "https://example.com/repo.git"},
		{"remote", "rename", "origin", "upstream"},
		{"remote", "set-url", "upstream", "https://example.com/other.git"},
		{"remote", "prune", "--dry-run", "upstream"},
		{"remote", "remove", "upstream"},
	}
	if !reflect.DeepEqual(runner.args, want) {
		t.Fatalf("remote lifecycle argv = %#v, want %#v", runner.args, want)
	}
}

func TestRemoteLifecycleRejectsUnsafeNamesAndURLs(t *testing.T) {
	var runner captureRemoteRunner
	if _, err := Add(context.Background(), &runner, "-origin", "https://example.com/repo.git"); !errors.Is(err, ErrInvalidRemoteName) {
		t.Fatalf("add unsafe name error = %v", err)
	}
	if _, err := Rename(context.Background(), &runner, "origin", "bad/name"); !errors.Is(err, ErrInvalidRemoteName) {
		t.Fatalf("rename unsafe name error = %v", err)
	}
	if _, err := SetURL(context.Background(), &runner, "origin", "https://example.com/bad\nurl"); !errors.Is(err, ErrMissingURL) {
		t.Fatalf("set-url unsafe URL error = %v", err)
	}
}

func TestGetURLRedactsCredentials(t *testing.T) {
	runner := captureRemoteRunner{results: []git.Result{{Stdout: []byte("https://alice:secret@example.com/repo.git\n")}}}
	got, err := GetURL(context.Background(), &runner, "origin", false)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || got == "https://alice:secret@example.com/repo.git" || strings.Contains(got, "secret") {
		t.Fatalf("get-url leaked credentials: %q", got)
	}
}

func TestRemoteOperationsRejectOptionLikeNames(t *testing.T) {
	_, err := Push(context.Background(), git.Runner{}, "-origin", "main", false)
	if !errors.Is(err, ErrMissingRemote) {
		t.Fatalf("expected remote validation error, got %v", err)
	}
}

func TestPushTagRequiresSafeExplicitTag(t *testing.T) {
	_, err := PushTag(context.Background(), git.Runner{}, "origin", "-v1")
	if !errors.Is(err, ErrMissingTag) {
		t.Fatalf("expected tag validation error, got %v", err)
	}
}

func TestDeleteTagRequiresExplicitRemoteAndSafeTag(t *testing.T) {
	_, err := DeleteTag(context.Background(), git.Runner{}, "", "v1")
	if !errors.Is(err, ErrMissingRemote) {
		t.Fatalf("expected remote validation error, got %v", err)
	}
	_, err = DeleteTag(context.Background(), git.Runner{}, "origin", "-v1")
	if !errors.Is(err, ErrMissingTag) {
		t.Fatalf("expected tag validation error, got %v", err)
	}
}

func TestPushSetUpstreamRejectsMissingRemote(t *testing.T) {
	_, err := PushSetUpstream(context.Background(), git.Runner{}, "", "main")
	if !errors.Is(err, ErrMissingRemote) {
		t.Fatalf("expected remote validation error, got %v", err)
	}
}

func TestParseRemoteSHA(t *testing.T) {
	if got := parseRemoteSHA([]byte("abc123\trefs/heads/main\n")); got != "abc123" {
		t.Fatalf("remote SHA = %q", got)
	}
	if got := parseRemoteSHA(nil); got != "" {
		t.Fatalf("empty remote SHA = %q", got)
	}
}

func TestPushAndDeleteTagAgainstBareRemote(t *testing.T) {
	ctx := context.Background()
	localDir := t.TempDir()
	remoteDir := t.TempDir()
	local := git.NewRunner(localDir)
	remote := git.NewRunner(remoteDir)

	runTestGit(t, remote, "init", "--bare")
	runTestGit(t, local, "init")
	runTestGit(t, local, "config", "user.name", "gitwatch test")
	runTestGit(t, local, "config", "user.email", "gitwatch@example.test")
	runTestGit(t, local, "config", "commit.gpgsign", "false")
	runTestGit(t, local, "config", "tag.gpgsign", "false")
	if err := os.WriteFile(localDir+"/README", []byte("test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runTestGit(t, local, "add", "README")
	runTestGit(t, local, "commit", "-m", "initial")
	runTestGit(t, local, "remote", "add", "origin", remoteDir)
	runTestGit(t, local, "tag", "v1")

	if _, err := PushTag(ctx, local, "origin", "v1"); err != nil {
		t.Fatalf("push tag: %v", err)
	}
	if _, err := remote.Run(ctx, "show-ref", "--verify", "refs/tags/v1"); err != nil {
		t.Fatalf("remote tag missing after push: %v", err)
	}
	if _, err := DeleteTag(ctx, local, "origin", "v1"); err != nil {
		t.Fatalf("delete remote tag: %v", err)
	}
	if _, err := remote.Run(ctx, "show-ref", "--verify", "refs/tags/v1"); err == nil {
		t.Fatal("remote tag still exists after explicit deletion")
	}
}

func runTestGit(t *testing.T, runner git.Runner, args ...string) {
	t.Helper()
	if _, err := runner.Run(context.Background(), args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}
