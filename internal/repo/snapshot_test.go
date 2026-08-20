package repo

import "testing"

func TestStatusLabel(t *testing.T) {
	if StatusLabel(Entry{Staged: true, Unstaged: true}) != "staged + modified" || StatusLabel(Entry{Conflicted: true}) != "conflict" {
		t.Fatal("status labels are not semantic")
	}
}

func TestConflictType(t *testing.T) {
	if (Entry{Conflicted: true, XY: "UU"}).ConflictType() != "both modified" {
		t.Fatal("conflict type")
	}
}
