package commands

import "testing"

func TestSearchRanksCompactMatchesAndKeepsDisabledActions(t *testing.T) {
	results := Search([]Action{{ID: "stage", Label: "Stage file", Enabled: true}, {ID: "status", Label: "Show status", Enabled: false, Reason: "no repository"}}, "stf")
	if len(results) != 1 || results[0].ID != "stage" {
		t.Fatalf("unexpected fuzzy matches: %#v", results)
	}
	results = Search([]Action{{ID: "status", Label: "Show status", Enabled: false, Reason: "no repository"}}, "")
	if len(results) != 1 || results[0].Enabled || results[0].Reason == "" {
		t.Fatalf("disabled action was lost: %#v", results)
	}
}

func TestSearchSupportsRepositoryAndCategoryPrefixes(t *testing.T) {
	actions := []Action{
		{ID: "repo-a", Label: "Open repository alpha", Category: "repository"},
		{ID: "status", Label: "Show status", Category: "workspace"},
		{ID: "repo-b", Label: "Open repository beta", Category: "repository"},
	}
	for _, query := range []string{"repo: beta", "repository: beta", "category:repository beta"} {
		results := Search(actions, query)
		if len(results) != 1 || results[0].ID != "repo-b" {
			t.Fatalf("Search(%q) = %#v", query, results)
		}
	}
	if got := Search(actions, "category:workspace"); len(got) != 1 || got[0].ID != "status" {
		t.Fatalf("category-only search = %#v", got)
	}
}
