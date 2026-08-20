package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func initRepo(t *testing.T, dir string) {
	t.Helper()
	if _, err := NewRunner(dir).Run(context.Background(), "init", "--", dir); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverNestedUnbornRepository(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := Discover(context.Background(), nested)
	if err != nil {
		t.Fatal(err)
	}
	if !samePath(d.Root, dir) || !d.Worktree || !d.Unborn || d.Detached || d.Bare {
		t.Fatalf("unexpected discovery: %+v", d)
	}
}

func TestDiscoverDetachedHead(t *testing.T) {
	dir := t.TempDir()
	initRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(dir)
	if _, err := r.Run(context.Background(), "add", "--", "file"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), "checkout", "--detach", "HEAD"); err != nil {
		t.Fatal(err)
	}
	d, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Detached || d.Unborn || !strings.HasPrefix(d.Head, "") {
		t.Fatalf("unexpected detached discovery: %+v", d)
	}
}
