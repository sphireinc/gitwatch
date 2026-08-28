package git

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadUnpushedUsesBoundedRangeArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-git")
	content := "#!/bin/sh\ncase \"$1\" in\nrev-parse) printf head123 ;;\nrev-list) printf 2 ;;\nlog) printf '* abc123 first\\n* def456 second\\n' ;;\nesac\nprintf '%s' \"$*\" > args\n"
	if err := os.WriteFile(script, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
	result, err := LoadUnpushed(context.Background(), Runner{Binary: script, Dir: dir}, "origin/main", 100)
	if err != nil || result.Count != 2 || result.Head != "head123" || len(result.Lines) != 2 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	args, err := os.ReadFile(filepath.Join(dir, "args"))
	if err != nil || !strings.Contains(string(args), "log --graph --decorate --oneline --no-color --max-count 100 origin/main..HEAD") {
		t.Fatalf("args=%q err=%v", args, err)
	}
}

func TestLoadUnpushedCapsLimit(t *testing.T) {
	if got, want := capUnpushedLimit(5000), MaxUnpushedCommits; got != want {
		t.Fatalf("limit=%d want=%d", got, want)
	}
}

func capUnpushedLimit(limit int) int {
	if limit > MaxUnpushedCommits {
		return MaxUnpushedCommits
	}
	return limit
}
