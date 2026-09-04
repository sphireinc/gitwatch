package recommend

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
)

func TestRecommendPolyglotMarkersAndReasons(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"go.mod", "package.json", "composer.json", "pyproject.toml", "Cargo.toml", "pom.xml", "app.csproj"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("marker"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	report, err := Recommend(root, cat, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Recommendations) < 7 {
		t.Fatalf("recommendations=%+v", report.Recommendations)
	}
	for _, rec := range report.Recommendations {
		if len(rec.Reasons) == 0 {
			t.Fatalf("reason missing for %+v", rec)
		}
	}
}

func TestRecommendIsBoundedAndDoesNotSelect(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 20; i++ {
		if err := os.WriteFile(filepath.Join(root, filepath.Base(filepath.Join("file", string(rune('a'+i)))+".go")), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cat, _ := catalog.Default()
	report, err := Recommend(root, cat, Options{MaxFiles: 3, MaxDepth: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Truncated || report.VisitedFiles > 4 {
		t.Fatalf("unbounded report=%+v", report)
	}
}

func TestMappedIDsExistInCatalog(t *testing.T) {
	cat, _ := catalog.Default()
	for name, id := range markerIDs {
		if _, ok := cat.Get(id); !ok {
			t.Fatalf("marker %s maps to missing %s", name, id)
		}
	}
	for ext, id := range extensionIDs {
		if _, ok := cat.Get(id); !ok {
			t.Fatalf("extension %s maps to missing %s", ext, id)
		}
	}
}
