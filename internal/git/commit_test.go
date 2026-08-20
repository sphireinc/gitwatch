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
