package history

import "testing"

func TestSelectionIsScopedAndRangeUsesGitApplicationOrder(t *testing.T) {
	selection, err := NewSelection("repo-a", "origin/main", 4)
	if err != nil {
		t.Fatal(err)
	}
	commits := []Commit{{SHA: "new"}, {SHA: "middle"}, {SHA: "old"}}
	selection, err = selection.SelectRange(commits, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	got := selection.SHAs()
	if len(got) != 3 || got[0] != "old" || got[1] != "middle" || got[2] != "new" {
		t.Fatalf("application order = %#v", got)
	}
	if err := selection.InScope("repo-b", "origin/main", 4); err == nil {
		t.Fatal("cross-repository selection was accepted")
	}
	if err := selection.InScope("repo-a", "origin/main", 4); err != nil {
		t.Fatalf("pagination generation was rejected: %v", err)
	}
}

func TestSelectionToggleAndClearPreserveScope(t *testing.T) {
	selection, err := NewSelection("repo", "main", 1)
	if err != nil {
		t.Fatal(err)
	}
	selection, err = selection.Toggle("abc")
	if err != nil || selection.Count() != 1 {
		t.Fatalf("toggle add = %#v err=%v", selection, err)
	}
	selection, err = selection.Toggle("abc")
	if err != nil || selection.Count() != 0 {
		t.Fatalf("toggle remove = %#v err=%v", selection, err)
	}
	cleared := selection.Clear()
	if cleared.Repository() != "repo" || cleared.Ref() != "main" || cleared.Generation() != 1 {
		t.Fatalf("clear changed scope = %#v", cleared)
	}
}
