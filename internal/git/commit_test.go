package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitUsesMessageStdin(t *testing.T) {
	dir := t.TempDir()
	r := NewRunner(dir)
	if _, e := r.Run(context.Background(), "init", "--", dir); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := r.Stage(context.Background(), []byte("a")); e != nil {
		t.Fatal(e)
	}
	out, e := r.Commit(context.Background(), CommitOptions{Message: []byte("message from stdin\n")})
	if e != nil {
		t.Fatal(e)
	}
	if len(strings.TrimSpace(out.SHA)) < 7 {
		t.Fatal(out.SHA)
	}
}

func TestCommitRejectsMultilineAuthor(t *testing.T) {
	_, err := Runner{Binary: "git", Dir: t.TempDir()}.Commit(context.Background(), CommitOptions{Message: []byte("subject\n"), Author: "Alice\n<alice@example.com>"})
	if err == nil || !strings.Contains(err.Error(), "author") {
		t.Fatalf("expected author validation error, got %v", err)
	}
}

func TestCommitConfigReadsOptionalSettings(t *testing.T) {
	dir := t.TempDir()
	runner := NewRunner(dir)
	if _, err := runner.Run(context.Background(), "init", "--", dir); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "config", "user.name", "Alice"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "config", "user.email", "alice@example.com"); err != nil {
		t.Fatal(err)
	}
	config := runner.CommitConfig(context.Background())
	if config.UserName != "Alice" || config.UserEmail != "alice@example.com" {
		t.Fatalf("unexpected config: %#v", config)
	}
}
