package match

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
)

func TestMatchUnmanagedFullPartialAndOverlap(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	template, ok := cat.Get("root/Node")
	if !ok {
		t.Fatal("missing Node")
	}
	doc, err := document.Parse(template.Content)
	if err != nil {
		t.Fatal(err)
	}
	results := Match(doc, cat)
	var node Result
	for _, result := range results {
		if result.TemplateID == "root/Node" {
			node = result
			break
		}
	}
	if node.Kind != domain.UnmanagedFull || node.Present != node.Total {
		t.Fatalf("node=%+v", node)
	}
	doc, _ = document.Parse([]byte("node_modules/\n"))
	results = Match(doc, cat)
	for _, result := range results {
		if result.TemplateID == "root/Node" {
			if result.Kind != domain.Partial {
				t.Fatalf("partial node=%+v", result)
			}
			break
		}
	}
	for _, result := range results {
		if result.Kind == domain.UnmanagedFull && result.TemplateID != "root/Node" {
			if result.Present == 0 {
				t.Errorf("full result has no coverage: %+v", result)
			}
			break
		}
	}
}

func TestMatchManagedCurrentOldAndEdited(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	template, ok := cat.Get("root/Node")
	if !ok {
		t.Fatal("missing Node")
	}
	encoded, err := managed.EncodeManagedBlock(template.ID, "github/gitignore", cat.Version(), template.ContentSHA256, template.Content, []byte("\n"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := document.Parse(encoded)
	if err != nil {
		t.Fatal(err)
	}
	results := Match(doc, cat)
	result := find(results, template.ID)
	if result.Kind != domain.ManagedExact || result.UpdateAvailable {
		t.Fatalf("current=%+v", result)
	}
	encoded, _ = managed.EncodeManagedBlock(template.ID, "github/gitignore", "old-commit", template.ContentSHA256, template.Content, []byte("\n"))
	doc, _ = document.Parse(encoded)
	result = find(Match(doc, cat), template.ID)
	if result.Kind != domain.ManagedExact || !result.UpdateAvailable {
		t.Fatalf("old=%+v", result)
	}
	encoded, _ = managed.EncodeManagedBlock(template.ID, "github/gitignore", cat.Version(), "edited", template.Content, []byte("\n"))
	doc, _ = document.Parse(encoded)
	result = find(Match(doc, cat), template.ID)
	if result.Kind != domain.ManagedEdited || result.Warning == "" {
		t.Fatalf("edited=%+v", result)
	}
}

func find(results []Result, id domain.TemplateID) Result {
	for _, result := range results {
		if result.TemplateID == id {
			return result
		}
	}
	return Result{}
}
