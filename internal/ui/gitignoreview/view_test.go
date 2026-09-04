package gitignoreview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
	"github.com/sphireinc/git-watch/internal/gitignore/recommend"
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

func TestPreviewIsVisibleAndSizeIsBounded(t *testing.T) {
	m := testModel(t)
	m.SetPreview("--- .gitignore (before)\n+++ .gitignore (after)\n# selected templates")
	m.SetSize(80, 24)
	if !strings.Contains(m.View(), "CREATE PREVIEW") {
		t.Fatal("creation preview missing")
	}
}

func TestViewSanitizesUntrustedTemplateAndPreviewText(t *testing.T) {
	m := RepositoryModel{
		RepositoryID: "repo\x1b[31m",
		Entries:      []Entry{{Template: catalog.Template{Template: domain.Template{ID: "root/Go", Name: "bad\x1b[31m", SourcePath: "x\x1b]8;;evil\a", Category: domain.CategoryRoot}, Content: []byte("rule\x1b[2J")}, Match: match.Result{Kind: domain.Absent}}},
		Selected:     0, Width: 80, Height: 24,
		PreviewText: "diff\x1b[31m\nnext",
	}
	view := m.View()
	if strings.Contains(view, "\x1b") || strings.Contains(view, "\a") {
		t.Fatalf("unsafe control reached view: %q", view)
	}
}

func BenchmarkFilterUpdates(b *testing.B) {
	m := testModelForBenchmark(b)
	queries := []string{"java", "python", "go", "", "community"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.SetQuery(queries[i%len(queries)])
	}
}

func testModelForBenchmark(b *testing.B) RepositoryModel {
	b.Helper()
	cat, err := catalog.Default()
	if err != nil {
		b.Fatal(err)
	}
	return New(domain.RepositoryID("benchmark"), cat, nil)
}

func TestRecommendationsExplainWithoutAutoSelecting(t *testing.T) {
	m := testModel(t)
	m.SetRecommendations([]recommend.Recommendation{{TemplateID: "root/CakePHP", Confidence: .9, Reasons: []string{"composer.json detected"}}})
	for _, entry := range m.AllEntries {
		if entry.Template.ID == "root/CakePHP" {
			if !entry.Recommended || entry.Selected || len(entry.Reasons) != 1 {
				t.Fatalf("recommendation state = %+v", entry)
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
