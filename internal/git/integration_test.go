package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	s, err = Snapshot(context.Background(), d, 2)
	if err != nil || len(s.Entries) != 1 || s.Entries[0].Staged || !s.Entries[0].Unstaged {
		t.Fatalf("post-mutation refresh state not observed: err=%v snapshot=%+v", err, s)
	}
}

func TestSnapshotDoesNotRewriteIndexStatCache(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := NewRunner(dir)
	for _, args := range [][]string{
		{"init", "--", dir},
		{"config", "user.name", "gitwatch"},
		{"config", "user.email", "gitwatch@example.com"},
	} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	tracked := filepath.Join(dir, "tracked.txt")
	if err := os.WriteFile(tracked, []byte("unchanged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, []byte("tracked.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "commit", "-m", "baseline"); err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(discovery.GitDir, "index")
	indexBefore, err := os.Stat(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	changedTime := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(tracked, changedTime, changedTime); err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(ctx, discovery, 1); err != nil {
		t.Fatal(err)
	}
	indexAfter, err := os.Stat(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !indexAfter.ModTime().Equal(indexBefore.ModTime()) || indexAfter.Size() != indexBefore.Size() {
		t.Fatalf("read-only snapshot rewrote index: before=%v/%d after=%v/%d", indexBefore.ModTime(), indexBefore.Size(), indexAfter.ModTime(), indexAfter.Size())
	}
}

func TestUnusualFilenamesAndRenamePreservePaths(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	r := NewRunner(dir)
	for _, args := range [][]string{{"init", "--", dir}, {"config", "user.name", "gitwatch"}, {"config", "user.email", "gitwatch@example.com"}, {"config", "core.autocrlf", "false"}} {
		if _, err := r.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{"space name.txt", "café.txt", "-leading.txt"}
	if runtime.GOOS != "windows" {
		// Quotes and tabs are valid Unix filename bytes but prohibited by the
		// Windows filesystem API. Exercise them wherever the platform permits.
		names = append(names, "quote\"name.txt", "tab\tname.txt")
	}
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
	renamed := "renamed café.txt"
	if runtime.GOOS != "windows" {
		renamed = "renamed \"café\".txt"
	}
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
