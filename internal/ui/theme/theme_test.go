package theme

import (
	"strings"
	"testing"
)

func TestColorlessThemeStillHasSemanticSymbols(t *testing.T) {
	r := New(Dark, true)
	if !r.Colorless || Symbol("conflict") != "!" || Symbol("untracked") != "?" {
		t.Fatal("semantic colorless theme lost status meaning")
	}
	if rendered := r.Selection.Render("selected"); strings.Contains(rendered, "\x1b[") {
		t.Fatalf("colorless selection contains terminal styles: %q", rendered)
	}
}
