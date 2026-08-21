package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManagerFilesystemEventsAndReconciliation(t *testing.T) {
	root := t.TempDir()
	manager, err := Start(context.Background(), root, RequestedAuto, time.Second, 20*time.Millisecond, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	if manager.Mode() != ModeFS {
		t.Fatalf("mode = %s", manager.Mode())
	}
	if err := os.WriteFile(filepath.Join(root, "changed"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-manager.Events():
		if event.Mode != ModeFS || event.Err != nil {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("filesystem manager did not emit")
	}
}

func TestManagerAutoFallsBackToPolling(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing")
	manager, warning := Start(context.Background(), root, RequestedAuto, 5*time.Millisecond, time.Second, time.Millisecond)
	if manager == nil || warning == nil || !strings.Contains(warning.Error(), "using polling") {
		t.Fatalf("manager=%v warning=%v", manager, warning)
	}
	t.Cleanup(func() {
		if err := manager.Close(); err != nil {
			t.Error(err)
		}
	})
	if manager.Mode() != ModePoll {
		t.Fatalf("mode = %s", manager.Mode())
	}
	select {
	case event := <-manager.Events():
		if event.Mode != ModePoll || event.Err == nil {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("polling fallback did not report traversal failure")
	}
}

func TestManagerForcedFilesystemFailure(t *testing.T) {
	manager, err := Start(context.Background(), filepath.Join(t.TempDir(), "missing"), RequestedFS, time.Second, time.Second, time.Millisecond)
	if manager != nil || err == nil {
		t.Fatalf("manager=%v err=%v", manager, err)
	}
}

func TestManagerCloseClosesEvents(t *testing.T) {
	manager, err := Start(context.Background(), t.TempDir(), RequestedPoll, time.Hour, time.Hour, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-manager.Events():
		if ok {
			t.Fatal("events remained open")
		}
	case <-time.After(time.Second):
		t.Fatal("manager did not close events")
	}
}

func TestConcurrentManagerCloseWaitsForShutdown(t *testing.T) {
	manager, err := Start(context.Background(), t.TempDir(), RequestedPoll, time.Hour, time.Hour, 0)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	go func() { results <- manager.Close() }()
	go func() { results <- manager.Close() }()
	for range 2 {
		select {
		case closeErr := <-results:
			if closeErr != nil {
				t.Fatal(closeErr)
			}
		case <-time.After(time.Second):
			t.Fatal("concurrent close did not wait for shutdown")
		}
	}
}

func TestManagerRuntimeFailureVisiblyFallsBackToPolling(t *testing.T) {
	manager, err := Start(context.Background(), t.TempDir(), RequestedAuto, time.Hour, time.Hour, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	manager.mu.Lock()
	filesystem := manager.fs
	manager.mu.Unlock()
	if err := filesystem.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-manager.Events():
		if event.Mode != ModePoll || event.Err == nil || !strings.Contains(event.Err.Error(), "using polling") {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("runtime watcher failure was not reported")
	}
	if manager.Mode() != ModePoll {
		t.Fatalf("mode = %s", manager.Mode())
	}
	if err := manager.Close(); err != nil {
		t.Fatal(err)
	}
}
