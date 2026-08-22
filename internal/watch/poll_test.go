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

func TestPollerReconcilesWithoutAChangedBoundedSignature(t *testing.T) {
	p := NewPoller(t.TempDir(), 5*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	select {
	case event := <-p.Events(ctx):
		if event.Err != nil || event.Mode != ModePoll {
			t.Fatalf("unexpected reconciliation event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("poller did not emit an unconditional reconciliation hint")
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

func TestPollerSignatureSeesExternalGitMetadata(t *testing.T) {
	root := t.TempDir()
	metadata := t.TempDir()
	p := NewPollerWithMetadata(root, []string{metadata}, time.Second)
	before, err := p.signature()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metadata, "HEAD"), []byte("ref: refs/heads/main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := p.signature()
	if err != nil {
		t.Fatal(err)
	}
	if after == before {
		t.Fatal("metadata change did not change polling signature")
	}
}

func TestPollerSkipsGitObjectTraversal(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	objectsDir := filepath.Join(gitDir, "objects")
	if err := os.MkdirAll(objectsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	objectPath := filepath.Join(objectsDir, "ignored")
	if err := os.WriteFile(objectPath, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	p := NewPollerWithMetadata(root, []string{gitDir}, time.Second)
	before, err := p.signature()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(objectPath, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Force the directory metadata change observed naturally on Windows so the
	// bounded-signature regression remains deterministic on every platform.
	changedTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(objectsDir, changedTime, changedTime); err != nil {
		t.Fatal(err)
	}
	afterObject, err := p.signature()
	if err != nil {
		t.Fatal(err)
	}
	if afterObject != before {
		t.Fatal("object payload changed bounded status signature")
	}
	if err := os.WriteFile(filepath.Join(gitDir, "index"), []byte("index"), 0o600); err != nil {
		t.Fatal(err)
	}
	afterIndex, err := p.signature()
	if err != nil {
		t.Fatal(err)
	}
	if afterIndex == afterObject {
		t.Fatal("index change did not change status signature")
	}
}
