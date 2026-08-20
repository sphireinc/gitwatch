package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStageUnstagePreservesWorkingTree(t *testing.T) {
	dir := t.TempDir()
	if _, err := NewRunner(dir).Run(context.Background(), "init", "--", dir); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(dir)
	path := []byte("-odd name.txt")
	full := filepath.Join(dir, string(path))
	if err := os.WriteFile(full, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Stage(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(full)
	if _, err := r.Unstage(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(full)
	if string(before) != string(after) {
		t.Fatal("unstage changed worktree")
	}
}

func TestStageAllAndUnstageAll(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(dir)
	if _, err := r.Run(context.Background(), "init", "--", dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "one"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.StageAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := r.UnstageAll(context.Background()); err != nil {
		t.Fatal(err)
	}
}
