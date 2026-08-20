package hunks

import (
	"github.com/jusanchez/gitwatch/internal/patch"
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
