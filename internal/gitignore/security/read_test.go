package security

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadDocumentRejectsNULAndOversizedContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(path, []byte("safe\x00bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadDocument(root, 1024); !errors.Is(err, ErrBinary) {
		t.Fatalf("NUL error = %v", err)
	}
	if err := os.WriteFile(path, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadDocument(root, 4); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("size error = %v", err)
	}
}

func TestTargetIsRepositoryRootBoundAndRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	if got, err := Target(root); err != nil || got != filepath.Join(root, ".gitignore") {
		t.Fatalf("target = %q, err=%v", got, err)
	}
	link := filepath.Join(root, ".gitignore")
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, err := ReadDocument(root, 1024); !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("symlink error = %v", err)
	}
}

func FuzzReadDocument(f *testing.F) {
	f.Add([]byte("*.log\n"))
	f.Add([]byte("# comment\r\n"))
	f.Fuzz(func(t *testing.T, content []byte) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), content, 0600); err != nil {
			t.Fatal(err)
		}
		_, _, _ = ReadDocument(root, 1<<20)
	})
}
