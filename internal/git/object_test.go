package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestShowPathRejectsUnresolvedRevisionExpressions(t *testing.T) {
	if _, err := NewRunner(t.TempDir()).ShowPath(context.Background(), "HEAD", []byte("file"), 100); err == nil {
		t.Fatal("unresolved revision was accepted")
	}
}

func TestShowPathReadsBoundedHistoricalContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "space name.txt")
	if err := os.WriteFile(path, []byte("historical content"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(root)
	runner.Env = []string{
		"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=commit.gpgsign", "GIT_CONFIG_VALUE_0=false",
		"GIT_AUTHOR_NAME=Test User", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test User", "GIT_COMMITTER_EMAIL=test@example.com",
	}
	for _, args := range [][]string{{"init"}, {"add", "--", "space name.txt"}, {"commit", "-m", "initial"}} {
		if _, err := runner.Run(context.Background(), args...); err != nil {
			t.Fatal(err)
		}
	}
	sha, err := runner.Run(context.Background(), "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	content, err := runner.ShowPath(context.Background(), string(sha.Stdout[:len(sha.Stdout)-1]), []byte("space name.txt"), 100)
	if err != nil || string(content) != "historical content" {
		t.Fatalf("content=%q err=%v", content, err)
	}
}
