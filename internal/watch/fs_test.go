package watch

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	gitrunner "github.com/sphireinc/git-watch/internal/git"
)

func TestWatcherDebouncesAndSeesCreatedDirectories(t *testing.T) {
	root := t.TempDir()
	w, err := New(root, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
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
	if runtime.GOOS == "windows" {
		return
	}
	drainFilesystemEvents(events, 25*time.Millisecond)
	if err := os.Chmod(filepath.Join(root, "one"), 0o755); err != nil {
		t.Fatal(err)
	}
	awaitFilesystemEvent(t, events)
}

func TestWatcherSeesExternalGitMetadataAndRecreatedDirectory(t *testing.T) {
	root := t.TempDir()
	metadataParent := t.TempDir()
	metadata := filepath.Join(metadataParent, "worktree-metadata")
	if err := os.MkdirAll(filepath.Join(metadata, "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := NewWithMetadata(root, []string{metadata}, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := w.Events(ctx)
	if err := os.WriteFile(filepath.Join(metadata, "HEAD"), []byte("ref: refs/heads/main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	awaitFilesystemEvent(t, events)

	if err := os.RemoveAll(metadata); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(metadata, "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metadata, "HEAD"), []byte("ref: refs/heads/topic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	awaitFilesystemEvent(t, events)
	if err := os.WriteFile(filepath.Join(metadata, "index"), []byte("index"), 0o600); err != nil {
		t.Fatal(err)
	}
	awaitFilesystemEvent(t, events)
	drainFilesystemEvents(events, 25*time.Millisecond)
	ref := filepath.Join(metadata, "refs", "heads", "topic")
	if err := os.WriteFile(ref, []byte("0123456789abcdef\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Err != nil || event.Mode != ModeFS || event.Path != ref {
			t.Fatalf("recreated ref event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("watcher did not restore nested metadata watches")
	}
}

func TestSnapshotReadDoesNotEmitMetadataHint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only CHMOD filtering covers kqueue metadata-read artifacts")
	}
	root := t.TempDir()
	runner := gitrunner.NewRunner(root)
	ctx := context.Background()
	for _, args := range [][]string{
		{"init", "--", root},
		{"config", "user.name", "gitwatch"},
		{"config", "user.email", "gitwatch@example.com"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	tracked := filepath.Join(root, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("baseline\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "add", "--", "tracked.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "commit", "-m", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tracked, []byte("modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitDir := filepath.Join(root, ".git")
	watcher, err := NewWithMetadata(root, []string{gitDir}, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = watcher.Close() })
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events := watcher.Events(watchCtx)
	if _, err := gitrunner.Snapshot(ctx, gitrunner.Discovery{Root: root, GitDir: gitDir, CommonDir: gitDir}, 1); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		t.Fatalf("read-only snapshot emitted watcher hint: %#v", event)
	case <-time.After(150 * time.Millisecond):
	}
}

func awaitFilesystemEvent(t *testing.T, events <-chan Event) {
	t.Helper()
	select {
	case event := <-events:
		if event.Err != nil || event.Mode != ModeFS {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("watcher did not emit")
	}
}

func drainFilesystemEvents(events <-chan Event, quiet time.Duration) {
	timer := time.NewTimer(quiet)
	defer timer.Stop()
	for {
		select {
		case <-events:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(quiet)
		case <-timer.C:
			return
		}
	}
}

func TestIsGitMetadata(t *testing.T) {
	if !IsGitMetadata("/tmp/repo/.git/index") || IsGitMetadata("/tmp/repo/.gitignore") {
		t.Fatal("metadata classification failed")
	}
}
