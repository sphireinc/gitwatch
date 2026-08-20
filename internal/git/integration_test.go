package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStatusMutationRefreshFlow(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(dir)
	if _, err := r.Run(context.Background(), "init", "--", dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(path, []byte("one\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Stage(context.Background(), []byte("tracked.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run(context.Background(), "-c", "user.name=gitwatch", "-c", "user.email=gitwatch@example.com", "commit", "-m", "initial"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two\n"), 0644); err != nil {
		t.Fatal(err)
	}
	d, err := Discover(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Snapshot(context.Background(), d, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Entries) != 1 || !s.Entries[0].Unstaged {
		t.Fatalf("unexpected modified snapshot: %+v", s)
	}
	if _, err := r.Stage(context.Background(), []byte("tracked.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Unstage(context.Background(), []byte("tracked.txt")); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s, err = Snapshot(context.Background(), d, 2)
		if err == nil && len(s.Entries) == 1 && !s.Entries[0].Staged && s.Entries[0].Unstaged {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("post-mutation refresh state not observed")
}
