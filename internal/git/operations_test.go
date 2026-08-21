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

func TestRestoreStagedAndWorkingTreeContent(t *testing.T) {
	dir := t.TempDir()
	runner := NewRunner(dir)
	ctx := context.Background()
	if _, err := runner.Run(ctx, "init", "--", dir); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "config", "user.name", "gitwatch test"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "config", "user.email", "gitwatch-test@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "config", "core.autocrlf", "false"); err != nil {
		t.Fatal(err)
	}
	path := []byte("tracked file.txt")
	full := filepath.Join(dir, string(path))
	if err := os.WriteFile(full, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, path); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "commit", "-m", "baseline"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("discard me\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(ctx, path); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Restore(ctx, path, true, true); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(full)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original\n" {
		t.Fatalf("restored content = %q", content)
	}
	status, err := runner.Run(ctx, "status", "--porcelain=v1", "-z")
	if err != nil {
		t.Fatal(err)
	}
	if len(status.Stdout) != 0 {
		t.Fatalf("worktree remains dirty: %q", status.Stdout)
	}
}
