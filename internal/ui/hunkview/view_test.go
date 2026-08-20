package hunkview

import (
	"github.com/jusanchez/gitwatch/internal/patch"
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
