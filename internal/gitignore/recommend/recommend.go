// Package recommend detects bounded repository signals and maps them to
// catalog templates. It never writes files or selects a template for a user.
package recommend

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

const (
	DefaultMaxFiles = 2000
	DefaultMaxDepth = 6
)

type Options struct{ MaxFiles, MaxDepth int }

type Recommendation struct {
	TemplateID domain.TemplateID
	Confidence float64
	Reasons    []string
}

type Report struct {
	Recommendations []Recommendation
	VisitedFiles    int
	SampledFiles    int
	Truncated       bool
}

type signal struct{ name, path string }

var markerIDs = map[string]domain.TemplateID{
	"go.mod": "root/Go", "package.json": "root/Node", "composer.json": "root/Composer",
	"pyproject.toml": "root/Python", "requirements.txt": "root/Python", "cargo.toml": "root/Rust",
	"pom.xml": "root/Java", "build.gradle": "root/Gradle", ".csproj": "root/Dotnet",
}

var extensionIDs = map[string]domain.TemplateID{
	".go": "root/Go", ".js": "root/Node", ".jsx": "root/Node", ".ts": "root/Node", ".tsx": "root/Node",
	".php": "root/Composer", ".py": "root/Python", ".rs": "root/Rust", ".java": "root/Java", ".cs": "root/Dotnet",
}

// Recommend performs a bounded walk. It skips dependency/build directories,
// caps visited files, and returns partial results with Truncated=true when a
// repository exceeds the budget.
func Recommend(root string, cat *catalog.Catalog, options Options) (Report, error) {
	if root == "" {
		return Report{}, fmt.Errorf("repository root is required")
	}
	if cat == nil {
		return Report{}, domain.ErrCatalogUnavailable
	}
	if options.MaxFiles <= 0 {
		options.MaxFiles = DefaultMaxFiles
	}
	if options.MaxDepth <= 0 {
		options.MaxDepth = DefaultMaxDepth
	}
	counts := map[domain.TemplateID]int{}
	reasons := map[domain.TemplateID][]string{}
	seenReasons := map[string]bool{}
	report := Report{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if path != root && depth(root, path) > options.MaxDepth {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			name := strings.ToLower(entry.Name())
			if path != root && (name == ".git" || name == "node_modules" || name == "vendor" || name == "target" || name == "dist" || name == "build") {
				return fs.SkipDir
			}
			return nil
		}
		report.VisitedFiles++
		if report.VisitedFiles > options.MaxFiles {
			report.Truncated = true
			return fs.SkipAll
		}
		base := strings.ToLower(entry.Name())
		if id, ok := markerIDs[base]; ok {
			addSignal(id, base+" detected", counts, reasons, seenReasons)
		}
		if strings.HasSuffix(base, ".csproj") {
			addSignal("root/Dotnet", entry.Name()+" detected", counts, reasons, seenReasons)
		}
		if id, ok := extensionIDs[strings.ToLower(filepath.Ext(base))]; ok {
			report.SampledFiles++
			addSignal(id, fmt.Sprintf("source files with %s extension detected", filepath.Ext(base)), counts, reasons, seenReasons)
		}
		return nil
	})
	if err != nil {
		return report, err
	}
	for id, count := range counts {
		if count == 0 {
			continue
		}
		if !catalogHas(cat, id) {
			continue
		}
		confidence := 0.45
		if len(reasons[id]) > 0 {
			confidence = 0.9
		}
		if report.SampledFiles > 0 && count > 0 {
			ratio := float64(count) / float64(report.SampledFiles)
			if ratio > confidence {
				confidence = ratio
			}
		}
		report.Recommendations = append(report.Recommendations, Recommendation{TemplateID: id, Confidence: confidence, Reasons: append([]string(nil), reasons[id]...)})
	}
	sort.Slice(report.Recommendations, func(i, j int) bool {
		return report.Recommendations[i].TemplateID < report.Recommendations[j].TemplateID
	})
	return report, nil
}

func addSignal(id domain.TemplateID, reason string, counts map[domain.TemplateID]int, reasons map[domain.TemplateID][]string, seen map[string]bool) {
	counts[id]++
	key := id.String() + "|" + reason
	if !seen[key] {
		reasons[id] = append(reasons[id], reason)
		seen[key] = true
	}
}

func catalogHas(cat *catalog.Catalog, id domain.TemplateID) bool { _, ok := cat.Get(id); return ok }

func depth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}
