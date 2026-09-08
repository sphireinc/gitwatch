package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyReversePatchToWorktreeAndIndexChecksBeforeMutation(t *testing.T) {
	root := t.TempDir()
	runner := NewRunner(root)
	for _, args := range [][]string{{"init", "--", root}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.test"}, {"config", "commit.gpgsign", "false"}} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("one\ntwo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Stage(context.Background(), []byte("file.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Commit(context.Background(), CommitOptions{Message: []byte("initial\n")}); err != nil {
		t.Fatal(err)
	}
	patch := []byte("diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -1,2 +1,2 @@\n one\n-two\n+changed\n")
	if err := os.WriteFile(path, []byte("one\nchanged\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.RunInput(context.Background(), patch, "apply", "--cached", "--whitespace=nowarn", "-"); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ApplyReversePatchToWorktreeAndIndex(context.Background(), PartialPatch{Patch: patch}); err != nil {
		t.Fatal(err)
	}
	status, err := runner.Run(context.Background(), "status", "--porcelain=v1", "--", "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(status.Stdout)) != "" {
		t.Fatalf("reversed patch left status: %q", status.Stdout)
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "one\ntwo\n" {
		t.Fatalf("content = %q, err=%v", content, err)
	}
}
