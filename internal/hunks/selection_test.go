package hunks

import (
	"github.com/jusanchez/gitwatch/internal/patch"
	"strings"
	"testing"
)

func TestStableSelection(t *testing.T) {
	files, _ := patch.Parse("diff --git a/a b/a\n@@ -1 +1 @@\n-old\n+new\n")
	s := New()
	s.SelectHunk(0, 0, files[0].Hunks[0])
	if s.Count() != 2 || !s.Valid(files) {
		t.Fatal(s)
	}
	files, _ = patch.Parse("diff --git a/a b/a\n@@ -1 +1 @@\n-old\n")
	s.Invalidate(files)
	if s.Count() != 1 {
		t.Fatal(s.Count())
	}
}

func TestBuildPatchRetainsContextAndSelectedChanges(t *testing.T) {
	files, err := patch.Parse("diff --git a/a b/a\n@@ -1,3 +1,3 @@\n one\n-old\n+new\n three\n")
	if err != nil {
		t.Fatal(err)
	}
	selection := New()
	selection.Toggle(ID{File: 0, Hunk: 0, Line: 2})
	result, err := selection.BuildPatch(files)
	if err != nil {
		t.Fatal(err)
	}
	text := string(result)
	if !strings.Contains(text, " one\n") || !strings.Contains(text, "+new\n") || strings.Contains(text, "-old\n") {
		t.Fatalf("unexpected partial patch: %q", text)
	}
}
