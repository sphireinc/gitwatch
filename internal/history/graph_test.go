package history

import "testing"

func TestBuildGraphKeepsMergeParentsInLanes(t *testing.T) {
	rows := BuildGraph([]Commit{
		{SHA: "merge", Parents: []string{"main", "side"}},
		{SHA: "main", Parents: []string{"root"}},
		{SHA: "side", Parents: []string{"root"}},
		{SHA: "root"},
	})
	if len(rows) != 4 || rows[0].Lane != 0 || rows[1].Lane != 0 || rows[2].Lane != 1 {
		t.Fatalf("unexpected graph rows: %#v", rows)
	}
}

func TestFilterMatchesMetadata(t *testing.T) {
	commits := []Commit{{SHA: "abc", Author: "Alice", Subject: "Fix parser"}, {SHA: "def", Author: "Bob", Subject: "Docs"}}
	if got := Filter(commits, "PARSER"); len(got) != 1 || got[0].SHA != "abc" {
		t.Fatalf("subject filter: %#v", got)
	}
}

func TestBuildGraphClassifiesDecorations(t *testing.T) {
	rows := BuildGraph([]Commit{{SHA: "abc", Refs: []string{"HEAD -> main", "tag: v1.0", "origin/main"}}})
	if len(rows) != 1 || !rows[0].Head || len(rows[0].Tags) != 1 || len(rows[0].Branches) != 2 {
		t.Fatalf("unexpected decorations: %#v", rows)
	}
}

func TestBuildGraphKeepsOctopusParentsDistinct(t *testing.T) {
	rows := BuildGraph([]Commit{
		{SHA: "octopus", Parents: []string{"one", "two", "three"}},
		{SHA: "one", Parents: []string{"root"}},
		{SHA: "two", Parents: []string{"root"}},
		{SHA: "three", Parents: []string{"root"}},
	})
	if len(rows) != 4 || rows[0].Lanes != 1 || rows[1].Lane != 0 || rows[2].Lane != 1 || rows[3].Lane != 2 {
		t.Fatalf("octopus lane topology = %#v", rows)
	}
}

func TestBuildGraphPagePreservesLaneContinuityAcrossPagination(t *testing.T) {
	first := []Commit{{SHA: "merge", Parents: []string{"main", "side"}}, {SHA: "main", Parents: []string{"root"}}}
	second := []Commit{{SHA: "side", Parents: []string{"root"}}, {SHA: "root"}}
	firstRows, cursor := BuildGraphPage(first, GraphCursor{})
	secondRows, _ := BuildGraphPage(second, cursor)
	if firstRows[1].Lane != 0 || secondRows[0].Lane != 1 || secondRows[1].Lane != 0 {
		t.Fatalf("page boundary lanes = first=%#v second=%#v", firstRows, secondRows)
	}
}
