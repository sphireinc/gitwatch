package remotes

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestPullRequiresExplicitStrategy(t *testing.T) {
	_, err := Pull(context.Background(), git.Runner{}, "origin", "main", "")
	if !errors.Is(err, ErrStrategyRequired) {
		t.Fatalf("expected explicit strategy error, got %v", err)
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
