package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
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
	s, err = Snapshot(context.Background(), d, 2)
	if err != nil || len(s.Entries) != 1 || s.Entries[0].Staged || !s.Entries[0].Unstaged {
		t.Fatalf("post-mutation refresh state not observed: err=%v snapshot=%+v", err, s)
	}
}

func TestUnusualFilenamesAndRenamePreservePaths(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	r := NewRunner(dir)
	for _, args := range [][]string{{"init", "--", dir}, {"config", "user.name", "gitwatch"}, {"config", "user.email", "gitwatch@example.com"}} {
		if _, err := r.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{"space name.txt", "café.txt", "quote\"name.txt", "-leading.txt"}
	for i, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(fmt.Sprintf("file %d\n", i)), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := r.Stage(ctx, []byte(name)); err != nil {
			t.Fatalf("stage %q: %v", name, err)
		}
	}
	if _, err := r.Commit(ctx, CommitOptions{Message: []byte("unusual names\n")}); err != nil {
		t.Fatal(err)
	}
	renamed := "renamed \"café\".txt"
	if _, err := r.Run(ctx, "mv", "--", names[1], renamed); err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := Snapshot(ctx, discovery, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entries) != 1 {
		t.Fatalf("rename snapshot entries = %#v", snapshot.Entries)
	}
	entry := snapshot.Entries[0]
	if !entry.Renamed || entry.Path.String() != renamed || entry.Original.String() != names[1] {
		t.Fatalf("rename paths were not preserved: %+v", entry)
	}
}
