package catalog

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

func TestDefaultCatalogLoadsAndRanksSearch(t *testing.T) {
	catalog, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Version() != "52f5a2bf5785a851e69936a6f5c54a734b828046" {
		t.Fatalf("version=%q", catalog.Version())
	}
	results := catalog.Search("php")
	if len(results) == 0 || results[0].ID != domain.TemplateID("root/CakePHP") {
		t.Fatalf("PHP results start with %+v", results[:min(3, len(results))])
	}
	for _, query := range []string{"osx", "nodejs", ".NET", "cpp", "objective-c"} {
		if len(catalog.Search(query)) == 0 {
			t.Errorf("%q returned no results", query)
		}
	}
	if len(catalog.ByCategory()[domain.CategoryRoot]) == 0 || len(catalog.ByCategory()[domain.CategoryGlobal]) == 0 || len(catalog.ByCategory()[domain.CategoryCommunity]) == 0 {
		t.Fatal("missing catalog category")
	}
	list := catalog.List()
	if len(list) < 300 {
		t.Fatalf("catalog size=%d", len(list))
	}
	list[0].Content[0] ^= 1
	fresh, _ := catalog.Get(list[0].ID)
	if list[0].Content[0] == fresh.Content[0] {
		t.Fatal("List returned mutable catalog content")
	}
}

func BenchmarkSearchFullCatalog(b *testing.B) {
	catalog, err := Default()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = catalog.Search("java")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
