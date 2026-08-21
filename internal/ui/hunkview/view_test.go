package hunkview

import (
	"github.com/sphireinc/git-watch/internal/patch"
	"strings"
	"testing"
)

func TestHunkViewSelection(t *testing.T) {
	files, _ := patch.Parse("diff --git a/a b/a\n@@ -1 +1 @@\n-old\n+new\n")
	m := New(files)
	m.Toggle()
	if m.Selection.Count() != 1 || !strings.Contains(m.View(), "selected 1") {
		t.Fatal(m.View())
	}
}

func TestHunkViewNavigationAndMouseLineSelection(t *testing.T) {
	files, err := patch.Parse("diff --git a/a b/a\n@@ -1 +1 @@\n-old\n+new\n@@ -4 +4 @@\n-four\n+five\n\ndiff --git a/b b/b\n@@ -1 +1 @@\n-left\n+right\n")
	if err != nil {
		t.Fatal(err)
	}
	m := New(files)
	m.MoveHunk(1)
	if m.File != 0 || m.Hunk != 1 || m.Line != 0 {
		t.Fatalf("next hunk = %#v", m)
	}
	m.MoveFile(1)
	if m.File != 1 || m.Hunk != 0 {
		t.Fatalf("next file = %#v", m)
	}
	if !m.SelectLine(1) || m.Selection.Count() != 1 {
		t.Fatalf("select line = %#v", m.Selection)
	}
	if m.SelectLine(2) || m.Selection.Count() != 1 {
		t.Fatalf("context line selection = %#v", m.Selection)
	}
}

func TestHunkViewViewportKeepsCurrentLineVisible(t *testing.T) {
	files, err := patch.Parse("diff --git a/a b/a\n@@ -1,5 +1,5 @@\n-a\n+b\n-c\n+d\n-e\n+f\n")
	if err != nil {
		t.Fatal(err)
	}
	m := New(files)
	m.SetHeight(2)
	m.Move(5)
	if m.Offset != 4 || m.LineAt(0) != 4 || !strings.Contains(m.View(), "+ f") {
		t.Fatalf("viewport = offset %d line %d view %q", m.Offset, m.Line, m.View())
	}
}
