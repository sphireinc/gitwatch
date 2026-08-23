package git

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadCommitTreeUsesBoundedGraphArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-git")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nif [ \"$1\" = \"rev-parse\" ]; then printf abc123; else printf '* abc123 Commit\\n| * def456 Other\\n'; fi\nprintf '%s' \"$*\" > args\n"), 0700); err != nil {
		t.Fatal(err)
	}
	runner := Runner{Binary: script, Dir: dir}
	result, err := LoadCommitTree(context.Background(), runner, 100)
	if err != nil || result.Head != "abc123" || len(result.Lines) != 2 {
		t.Fatalf("tree=%#v err=%v", result, err)
	}
	args, err := os.ReadFile(filepath.Join(dir, "args"))
	if err != nil || !strings.Contains(string(args), "log --oneline --graph --decorate --all -n 100") {
		t.Fatalf("arguments=%q err=%v", args, err)
	}
}

func TestCommitTreeLimitIsCapped(t *testing.T) {
	if got := capCommitTreeLimit(5000); got != CommitTreeMaxCommits {
		t.Fatalf("limit=%d", got)
	}
}

func capCommitTreeLimit(limit int) int {
	if limit > CommitTreeMaxCommits {
		return CommitTreeMaxCommits
	}
	return limit
}
