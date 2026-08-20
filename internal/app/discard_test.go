package app

import "testing"

func TestPartialDiscardHighFriction(t *testing.T) {
	d := PartialDiscard{Path: "file", HunkCount: 1, LineCount: 2, Confirmed: true}
	if d.Accept("yes") || !d.Accept("discard") {
		t.Fatal(d.Prompt())
	}
}
