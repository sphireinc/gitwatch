package submodules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/git"
)

func TestParseConfigPreservesPathsAndRedactsURLCredentials(t *testing.T) {
	modules, err := parseConfig([]byte("submodule.alpha.path\nnested path\x00submodule.alpha.url\nhttps://alice:secret@example.test/repo\x00submodule.beta.path\n-odd\x00submodule.beta.url\ngit@example.test:repo\x00"))
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 2 {
		t.Fatalf("modules = %+v", modules)
	}
	if modules[0].Name != "alpha" || modules[0].Path != "nested path" || modules[0].URL != "https://example.test/repo" {
		t.Fatalf("alpha = %+v", modules[0])
	}
	if modules[1].Name != "beta" || modules[1].Path != "-odd" || modules[1].URL != "example.test:repo" {
		t.Fatalf("beta = %+v", modules[1])
	}
}

func TestParseStatusHandlesSpacesAndHealthMarkers(t *testing.T) {
	status, err := parseStatus([]byte(" 0123456789012345678901234567890123456789 nested path\n+fedcba9876543210fedcba9876543210fedcba98 dirty path (new commits)\n-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa uninitialized path (uninitialized)\n"), 8)
	if err != nil {
		t.Fatal(err)
	}
	if status["nested path"].State != StateClean || status["nested path"].CheckedOutCommit != "0123456789012345678901234567890123456789" {
		t.Fatalf("clean status = %+v", status["nested path"])
	}
	if status["dirty path"].State != StateDiverged || status["dirty path"].Divergence != 0 {
		t.Fatalf("dirty status = %+v", status["dirty path"])
	}
	if status["uninitialized path"].State != StateUninitialized {
		t.Fatalf("uninitialized status = %+v", status["uninitialized path"])
	}
}

func TestParseStatusHonorsDepthBound(t *testing.T) {
	status, err := parseStatus([]byte(" 0123456789012345678901234567890123456789 one/two/three\n"), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 0 {
		t.Fatalf("bounded status = %+v", status)
	}
}

func TestLoadRealRepositoryUsesGitConfigAndGitlinkStatus(t *testing.T) {
	ctx := context.Background()
	child := t.TempDir()
	childRunner := git.NewRunner(child)
	gitMustRun(t, ctx, childRunner, "init", "-b", "main", "--", child)
	if err := os.WriteFile(filepath.Join(child, "README"), []byte("child\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitMustRun(t, ctx, childRunner, "add", "--", "README")
	gitMustRun(t, ctx, childRunner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "child")
	childHead, err := childRunner.Run(ctx, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	parentRunner := git.NewRunner(parent)
	gitMustRun(t, ctx, parentRunner, "init", "-b", "main", "--", parent)
	gitMustRun(t, ctx, parentRunner, "config", "-f", ".gitmodules", "submodule.alpha.path", "nested path")
	gitMustRun(t, ctx, parentRunner, "config", "-f", ".gitmodules", "submodule.alpha.url", "https://user:secret@example.test/child")
	gitMustRun(t, ctx, parentRunner, "add", "--", ".gitmodules")
	gitMustRun(t, ctx, parentRunner, "update-index", "--add", "--cacheinfo", "160000,"+string(childHead.Stdout[:40])+",nested path")
	gitMustRun(t, ctx, parentRunner, "-c", "commit.gpgsign=false", "-c", "user.name=test", "-c", "user.email=test@example.test", "commit", "-m", "parent")
	snapshot, err := Load(ctx, parentRunner, LoadRequest{Repository: parent, Limits: Limits{MaxOutputBytes: 64 << 10}})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Modules) != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
	module := snapshot.Modules[0]
	if module.Path != "nested path" || module.RecordedCommit != string(childHead.Stdout[:40]) || module.State != StateUninitialized {
		t.Fatalf("module = %+v", module)
	}
	if module.URL != "https://example.test/child" {
		t.Fatalf("redacted URL = %q", module.URL)
	}
}

func gitMustRun(t *testing.T, ctx context.Context, runner git.Runner, args ...string) {
	t.Helper()
	if _, err := runner.Run(ctx, args...); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}
