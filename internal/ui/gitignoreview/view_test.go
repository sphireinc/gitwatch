package gitignoreview

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
)

func testModel(t *testing.T) RepositoryModel {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	return New(domain.RepositoryID("repo-a"), cat, []match.Result{{TemplateID: "root/Go", Kind: domain.ManagedExact, Present: 10, Total: 10}, {TemplateID: "root/CakePHP", Kind: domain.Partial, Present: 1, Total: 3}})
}

func TestSearchFindsPHPAndPreservesSelectionAcrossFilters(t *testing.T) {
	m := testModel(t)
	m.SetQuery("php")
	if len(m.Entries) == 0 || m.Entries[0].Template.Name != "CakePHP" {
		t.Fatalf("php results = %+v", m.Entries[:min(3, len(m.Entries))])
	}
	m.Toggle()
	m.SetQuery("")
	found := false
	for _, entry := range m.AllEntries {
		if entry.Template.Name == "CakePHP" && entry.Selected {
			found = true
		}
	}
	if !found {
		t.Fatal("selection did not survive query change")
	}
}

func TestFullMatchesPinnedAndIndicatorsAreSemantic(t *testing.T) {
	m := testModel(t)
	if len(m.Entries) < 2 || !m.Entries[0].Match.Kind.Full() {
		t.Fatalf("full match was not pinned: %+v", m.Entries[:2])
	}
	if indicator(m.Entries[0]) != "*" {
		t.Fatal("full match indicator is not *")
	}
	m.Selected = 1
	if indicator(m.Entries[1]) != "~" {
		t.Fatalf("partial indicator = %q", indicator(m.Entries[1]))
	}
}

func TestRepositoryScopeAndMouseParity(t *testing.T) {
	m := testModel(t)
	other := testModel(t)
	other.RepositoryID = "repo-b"
	m.SetQuery("go")
	if !m.Click(1, 4) || len(m.SelectedEntries()) != 1 {
		t.Fatal("row click did not select")
	}
	if len(other.SelectedEntries()) != 0 {
		t.Fatal("selection leaked between repositories")
	}
	if m.UpdateKey("space") != true {
		t.Fatal("space was not consumed")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
