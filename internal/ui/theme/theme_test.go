package theme

import "testing"

func TestColorlessThemeStillHasSemanticSymbols(t *testing.T) {
	r := New(Dark, true)
	if !r.Colorless || Symbol("conflict") != "!" || Symbol("untracked") != "?" {
		t.Fatal("semantic colorless theme lost status meaning")
	}
}
