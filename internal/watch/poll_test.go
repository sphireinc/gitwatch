package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSelectMode(t *testing.T) {
	if mode, _ := SelectMode(RequestedAuto, false); mode != ModePoll {
		t.Fatal(mode)
	}
	if mode, _ := SelectMode(RequestedAuto, true); mode != ModeFS {
		t.Fatal(mode)
	}
	if mode, _ := SelectMode(RequestedPoll, true); mode != ModePoll {
		t.Fatal(mode)
	}
	if _, err := SelectMode(RequestedFS, false); err == nil {
		t.Fatal("expected forced-fs failure")
	}
}

func TestPollerEmitsAfterChange(t *testing.T) {
	root := t.TempDir()
	p := NewPoller(root, 10*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := p.Events(ctx)
	time.Sleep(15 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(root, "change"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-events:
		if event.Mode != ModePoll {
			t.Fatal(event.Mode)
		}
	case <-time.After(time.Second):
		t.Fatal("poller did not emit")
	}
}

func TestPollerReportsTraversalErrors(t *testing.T) {
	p := NewPoller(filepath.Join(t.TempDir(), "missing"), time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	select {
	case event := <-p.Events(ctx):
		if event.Err == nil || event.Mode != ModePoll {
			t.Fatalf("unexpected event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("poller did not report traversal error")
	}
}
