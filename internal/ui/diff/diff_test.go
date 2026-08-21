package diff

import (
	"context"
	"github.com/sphireinc/git-watch/internal/git"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestViewerLoadsDiffAsynchronously(t *testing.T) {
	dir := t.TempDir()
	r := git.NewRunner(dir)
	if _, err := r.Run(context.Background(), "init", "--", dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("one\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Stage(context.Background(), []byte("file")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), []byte("two\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var v Viewer
	v.Open(context.Background(), r, []byte("file"), Unstaged)
	deadline := time.After(time.Second)
	for {
		result, loading := v.Snapshot()
		if !loading {
			if result.Err != nil || string(result.Text) == "" {
				t.Fatal(result.Err, string(result.Text))
			}
			break
		}
		select {
		case <-deadline:
			t.Fatal("diff remained loading")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
