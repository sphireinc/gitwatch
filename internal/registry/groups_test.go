package registry

import "testing"

func TestGroupsFavoritesAndOrdering(t *testing.T) {
	repositories := []Repository{{Path: "/a", Name: "a", Groups: []string{"work", "oss"}}, {Path: "/b", Name: "b", Groups: []string{"work"}}}
	if got := Groups(repositories); len(got) != 2 || got[0] != "oss" {
		t.Fatalf("unexpected groups: %#v", got)
	}
	updated := SetFavorite(repositories, "/b", true)
	if updated[0].Path != "/b" || repositories[1].Favorite {
		t.Fatalf("favorite update mutated or failed ordering: %#v", updated)
	}
	if got := InGroup(updated, "work"); len(got) != 2 {
		t.Fatalf("unexpected group filter: %#v", got)
	}
}
