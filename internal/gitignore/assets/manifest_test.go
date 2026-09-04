package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

type manifest struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Templates  []struct {
		ID         domain.TemplateID       `json:"id"`
		SourcePath string                  `json:"source_path"`
		Category   domain.TemplateCategory `json:"category"`
		SHA256     string                  `json:"sha256"`
		Bytes      int                     `json:"bytes"`
	} `json:"templates"`
}

func TestCheckedInManifestMatchesAssets(t *testing.T) {
	data, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var got manifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Repository != "github/gitignore" || len(got.Commit) != 40 {
		t.Fatalf("bad provenance: %+v", got)
	}
	ids := make([]string, len(got.Templates))
	categories := map[domain.TemplateCategory]bool{}
	for i, entry := range got.Templates {
		ids[i] = entry.ID.String()
		categories[entry.Category] = true
		if entry.SHA256 == "" || entry.Bytes < 0 {
			t.Fatalf("bad manifest entry: %+v", entry)
		}
		content, err := os.ReadFile(filepath.FromSlash("catalog/" + entry.ID.String() + ".gitignore"))
		if err != nil {
			t.Fatalf("%s: %v", entry.ID, err)
		}
		if len(content) != entry.Bytes {
			t.Fatalf("%s byte length=%d, manifest=%d", entry.ID, len(content), entry.Bytes)
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != entry.SHA256 {
			t.Fatalf("%s hash mismatch", entry.ID)
		}
		if entry.SourcePath == "" {
			t.Fatalf("%s has no source path", entry.ID)
		}
	}
	if !sort.StringsAreSorted(ids) {
		t.Fatal("manifest entries are not sorted by stable ID")
	}
	for _, category := range []domain.TemplateCategory{domain.CategoryRoot, domain.CategoryGlobal, domain.CategoryCommunity} {
		if !categories[category] {
			t.Fatalf("missing category %q", category)
		}
	}
}
