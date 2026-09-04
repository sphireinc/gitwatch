// Package catalog exposes the embedded, offline gitignore template catalog.
package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/sphireinc/git-watch/internal/gitignore/assets"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

type Template struct {
	domain.Template
	Content []byte
	Aliases []string
}

type Catalog struct {
	version string
	entries []Template
	byID    map[domain.TemplateID]int
}

type manifest struct {
	Repository string          `json:"repository"`
	Commit     string          `json:"commit"`
	Templates  []manifestEntry `json:"templates"`
}
type manifestEntry struct {
	ID         domain.TemplateID       `json:"id"`
	SourcePath string                  `json:"source_path"`
	Category   domain.TemplateCategory `json:"category"`
	SHA256     string                  `json:"sha256"`
	Bytes      int                     `json:"bytes"`
}

var (
	defaultOnce    sync.Once
	defaultCatalog *Catalog
	defaultErr     error
)

// Default loads and validates the embedded catalog once.
func Default() (*Catalog, error) {
	defaultOnce.Do(func() { defaultCatalog, defaultErr = load(assets.Files) })
	return defaultCatalog, defaultErr
}

func load(files interface{ ReadFile(string) ([]byte, error) }) (*Catalog, error) {
	manifestBytes, err := files.ReadFile("manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read catalog manifest: %w", err)
	}
	var raw manifest
	if err := json.Unmarshal(manifestBytes, &raw); err != nil {
		return nil, fmt.Errorf("decode catalog manifest: %w", err)
	}
	if raw.Repository != "github/gitignore" || len(raw.Commit) != 40 {
		return nil, fmt.Errorf("invalid catalog provenance")
	}
	entries := make([]Template, 0, len(raw.Templates))
	byID := make(map[domain.TemplateID]int, len(raw.Templates))
	for _, item := range raw.Templates {
		if _, exists := byID[item.ID]; exists {
			return nil, fmt.Errorf("duplicate template ID %q", item.ID)
		}
		content, err := files.ReadFile(filepath.ToSlash(filepath.Join("catalog", item.ID.String()+".gitignore")))
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", item.ID, err)
		}
		sum := sha256.Sum256(content)
		if hex.EncodeToString(sum[:]) != item.SHA256 || len(content) != item.Bytes {
			return nil, fmt.Errorf("catalog hash/length mismatch for %s", item.ID)
		}
		template := Template{Template: domain.Template{ID: item.ID, Name: strings.TrimSuffix(filepath.Base(item.SourcePath), ".gitignore"), Category: item.Category, SourcePath: item.SourcePath, ContentSHA256: item.SHA256}, Content: append([]byte(nil), content...), Aliases: aliases(item.ID)}
		byID[item.ID] = len(entries)
		entries = append(entries, template)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	byID = make(map[domain.TemplateID]int, len(entries))
	for i := range entries {
		byID[entries[i].ID] = i
	}
	return &Catalog{version: raw.Commit, entries: entries, byID: byID}, nil
}

func (c *Catalog) Version() string {
	if c == nil {
		return ""
	}
	return c.version
}

func (c *Catalog) List() []Template {
	if c == nil {
		return nil
	}
	return cloneTemplates(c.entries)
}

func (c *Catalog) Get(id domain.TemplateID) (Template, bool) {
	if c == nil {
		return Template{}, false
	}
	i, ok := c.byID[id]
	if !ok {
		return Template{}, false
	}
	return cloneTemplate(c.entries[i]), true
}

func (c *Catalog) ByCategory() map[domain.TemplateCategory][]Template {
	out := map[domain.TemplateCategory][]Template{}
	if c == nil {
		return out
	}
	for _, entry := range c.entries {
		out[entry.Category] = append(out[entry.Category], cloneTemplate(entry))
	}
	return out
}

type result struct {
	template Template
	score    int
}

// Search ranks exact/prefix name matches before aliases, path/category matches,
// then uses stable ID order as the final tie-breaker.
func (c *Catalog) Search(query string) []Template {
	if c == nil {
		return nil
	}
	q := normalize(query)
	if q == "" {
		return c.List()
	}
	results := make([]result, 0, len(c.entries))
	for _, entry := range c.entries {
		score := scoreTemplate(entry, q)
		if score > 0 {
			results = append(results, result{template: entry, score: score})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].template.ID < results[j].template.ID
	})
	out := make([]Template, len(results))
	for i := range results {
		out[i] = cloneTemplate(results[i].template)
	}
	return out
}

func scoreTemplate(entry Template, q string) int {
	name := normalize(entry.Name)
	id := normalize(entry.ID.String())
	source := normalize(entry.SourcePath)
	switch {
	case name == q:
		return 1000 + categoryBonus(entry.Category)
	case strings.HasPrefix(name, q):
		return 850 + categoryBonus(entry.Category)
	case name != "" && strings.Contains(name, q):
		return 700 + categoryBonus(entry.Category)
	}
	for _, alias := range entry.Aliases {
		a := normalize(alias)
		if a == q {
			return 800 + categoryBonus(entry.Category)
		}
		if strings.HasPrefix(a, q) {
			return 650 + categoryBonus(entry.Category)
		}
		if strings.Contains(a, q) {
			return 550 + categoryBonus(entry.Category)
		}
	}
	if strings.Contains(id, q) || strings.Contains(source, q) {
		return 450 + categoryBonus(entry.Category)
	}
	if strings.Contains(normalize(string(entry.Category)), q) {
		return 250 + categoryBonus(entry.Category)
	}
	return 0
}

func categoryBonus(category domain.TemplateCategory) int {
	if category == domain.CategoryRoot {
		return 3
	}
	return 0
}

func aliases(id domain.TemplateID) []string {
	known := map[string][]string{"global/macos": {"OSX"}, "root/node": {"NodeJS"}, "root/dotnet": {".NET", "dot net"}, "root/c++": {"cpp", "cplusplus"}, "root/objective-c": {"objc", "objective c"}}
	return append([]string(nil), known[strings.ToLower(id.String())]...)
}

func normalize(value string) string {
	value = strings.ToLower(value)
	replacements := []string{"nodejs", "node", "osx", "macos", "objective-c", "objectivec", "c++", "cpp", ".net", "dotnet"}
	for i := 0; i < len(replacements); i += 2 {
		value = strings.ReplaceAll(value, replacements[i], replacements[i+1])
	}
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cloneTemplate(in Template) Template {
	in.Content = append([]byte(nil), in.Content...)
	in.Aliases = append([]string(nil), in.Aliases...)
	return in
}
func cloneTemplates(in []Template) []Template {
	out := make([]Template, len(in))
	for i := range in {
		out[i] = cloneTemplate(in[i])
	}
	return out
}
