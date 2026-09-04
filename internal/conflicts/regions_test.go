package conflicts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseAndApplyMultipleCRLFRegions(t *testing.T) {
	data := []byte("before\r\n<<<<<<< ours\r\none\r\n=======\r\ntwo\r\n>>>>>>> theirs\r\nmiddle\r\n<<<<<<< ours\r\nthree\r\n=======\r\nfour\r\n>>>>>>> theirs\r\nafter\r\n")
	doc, err := ParseRegions(data, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Regions) != 2 || string(doc.Regions[0].Ours) != "one\r\n" {
		t.Fatalf("regions = %#v", doc.Regions)
	}
	updated, err := doc.Apply(0, ChoiceTheirs, nil, data)
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != "before\r\ntwo\r\nmiddle\r\n<<<<<<< ours\r\nthree\r\n=======\r\nfour\r\n>>>>>>> theirs\r\nafter\r\n" {
		t.Fatalf("updated = %q", updated)
	}
}

func TestApplyRejectsStaleExternalEditAndUndoIsBounded(t *testing.T) {
	doc, err := ParseRegions([]byte("<<<<<<< ours\na\n=======\nb\n>>>>>>> theirs\n"), 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.Apply(0, ChoiceOurs, nil, []byte("external\n")); err != ErrStaleDocument {
		t.Fatalf("stale error = %v", err)
	}
	undo := NewUndoStack(2)
	undo.Push([]byte("one"))
	undo.Push([]byte("two"))
	undo.Push([]byte("three"))
	if got, _ := undo.Undo(); string(got) != "three" {
		t.Fatalf("undo = %q", got)
	}
	if got, _ := undo.Undo(); string(got) != "two" {
		t.Fatalf("undo = %q", got)
	}
	if _, ok := undo.Undo(); ok {
		t.Fatal("undo exceeded bound")
	}
}

func TestAtomicWritePreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	if err := os.WriteFile(path, []byte("old"), 0o751); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(path, []byte("new"), info.Mode()); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o751 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
}
