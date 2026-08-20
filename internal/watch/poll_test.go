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
