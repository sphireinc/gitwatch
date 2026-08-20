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
