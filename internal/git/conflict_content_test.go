package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/conflicts"
)

func TestLoadConflictContentUsesStagesAndBoundsResult(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "result"), []byte("a\x00binary"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := NewRunner(dir)
	if _, err := runner.Run(context.Background(), "init", "--quiet"); err != nil {
		t.Fatal(err)
	}
	object, err := runner.RunInput(context.Background(), []byte("0123456789"), "hash-object", "-w", "--stdin")
	if err != nil {
		t.Fatal(err)
	}
	conflict := conflicts.Conflict{Path: []byte("result"), Base: conflicts.Stage{OID: string(object.Stdout[:len(object.Stdout)-1])}}
	bounded, boundedErr := runner.RunBounded(context.Background(), 4, "cat-file", "blob", conflict.Base.OID)
	if boundedErr == nil || len(bounded.Stdout) != 4 {
		t.Fatalf("bounded runner failed: len=%d err=%v", len(bounded.Stdout), boundedErr)
	}
	got, err := LoadConflictContent(context.Background(), runner, conflict, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Base.Truncated || string(got.Base.Bytes) != "0123" || !got.Result.Binary {
		t.Fatalf("bounded/content classification failed: %+v", got)
	}
	if !got.Ours.Missing || !got.Theirs.Missing {
		t.Fatalf("missing stages not explicit: %+v", got)
	}
}

func TestClassifyContent(t *testing.T) {
	var bounded boundedBuffer = boundedBuffer{limit: 4}
	if _, err := bounded.Write([]byte("0123456789")); err != nil || string(bounded.data) != "0123" || !bounded.over {
		t.Fatalf("bounded buffer failed: %+v", bounded)
	}
	if !classifyContent(Content{Bytes: []byte{1, 2, 0}}).Binary {
		t.Fatal("NUL content should be binary")
	}
	if !classifyContent(Content{Bytes: []byte{0xff}}).InvalidUTF8 {
		t.Fatal("invalid UTF-8 should be marked")
	}
}

func TestValidObjectID(t *testing.T) {
	if !validObjectID("0123456789012345678901234567890123456789") || validObjectID("HEAD:file") {
		t.Fatal("object ID validation failed")
	}
}
