package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatcherDebouncesAndSeesCreatedDirectories(t *testing.T) {
	root := t.TempDir()
	w, err := New(root, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := w.Events(ctx)
	if err := os.WriteFile(filepath.Join(root, "one"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "new", "two"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Err != nil {
			t.Fatal(event.Err)
		}
	case <-time.After(time.Second):
		t.Fatal("watcher did not emit")
	}
}

func TestIsGitMetadata(t *testing.T) {
	if !IsGitMetadata("/tmp/repo/.git/index") || IsGitMetadata("/tmp/repo/.gitignore") {
		t.Fatal("metadata classification failed")
	}
}
