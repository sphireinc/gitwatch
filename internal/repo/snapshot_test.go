package repo

import "testing"

func TestStatusLabel(t *testing.T) {
	if StatusLabel(Entry{Staged: true, Unstaged: true}) != "staged + modified" || StatusLabel(Entry{Conflicted: true}) != "conflict" {
		t.Fatal("status labels are not semantic")
	}
}
