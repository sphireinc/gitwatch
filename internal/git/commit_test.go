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
	for _, args := range [][]string{
		{"init", "--", dir},
		{"config", "user.name", "gitwatch test"},
		{"config", "user.email", "gitwatch-test@example.com"},
	} {
		if _, err := r.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
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

func TestCommitFixupUsesTypedTargetAndLeavesStagingExplicit(t *testing.T) {
	dir := t.TempDir()
	runner := NewRunner(dir)
	for _, args := range [][]string{
		{"init", "--", dir},
		{"config", "user.name", "gitwatch test"},
		{"config", "user.email", "gitwatch-test@example.com"},
	} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(context.Background(), []byte("a")); err != nil {
		t.Fatal(err)
	}
	base, err := runner.Commit(context.Background(), CommitOptions{Message: []byte("base\n")})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("one\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(context.Background(), []byte("a")); err != nil {
		t.Fatal(err)
	}
	fixup, err := runner.Commit(context.Background(), CommitOptions{FixupSHA: base.SHA})
	if err != nil {
		t.Fatal(err)
	}
	message, err := runner.Run(context.Background(), "show", "-s", "--format=%s", fixup.SHA)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(message.Stdout)); got != "fixup! base" {
		t.Fatalf("fixup subject = %q", got)
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
