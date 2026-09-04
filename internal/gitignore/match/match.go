// Package match compares lossless gitignore documents with catalog templates.
package match

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
)

type Result struct {
	TemplateID      domain.TemplateID
	Kind            domain.MatchKind
	Present         int
	Total           int
	UpdateAvailable bool
	Warning         string
}

// Match performs one document rule index pass, then evaluates every template
// against it. Rule spelling, escapes, and negation prefixes remain exact.
func Match(doc document.Document, cat *catalog.Catalog) []Result {
	if cat == nil {
		return nil
	}
	rules := make(map[string]struct{}, len(doc.Lines))
	for _, line := range doc.Lines {
		if line.Kind == document.Rule {
			rules[string(line.Text)] = struct{}{}
		}
	}
	blocks := managedBlocks(doc)
	results := make([]Result, 0, len(cat.List()))
	for _, template := range cat.List() {
		significant := ruleSet(template.Content)
		result := Result{TemplateID: template.ID, Total: len(significant)}
		if block, ok := blocks[template.ID]; ok {
			result.Present = len(significant)
			if block.ContentSHA256 == template.ContentSHA256 {
				result.Kind = domain.ManagedExact
			} else {
				result.Kind, result.Warning = domain.ManagedEdited, "managed block content hash differs from bundled template"
			}
			result.UpdateAvailable = block.Commit != "" && block.Commit != cat.Version()
			results = append(results, result)
			continue
		}
		for rule := range significant {
			if _, ok := rules[rule]; ok {
				result.Present++
			}
		}
		switch {
		case result.Total == 0:
			result.Kind = domain.Absent
		case result.Present == result.Total:
			result.Kind = domain.UnmanagedFull
		case result.Present > 0:
			result.Kind = domain.Partial
		default:
			result.Kind = domain.Absent
		}
		results = append(results, result)
	}
	return results
}

func ruleSet(content []byte) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range bytes.Split(content, []byte("\n")) {
		line = bytes.TrimSuffix(line, []byte("\r"))
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "\\#") {
			if strings.HasPrefix(trimmed, "\\#") {
				out[string(line)] = struct{}{}
			}
			continue
		}
		out[string(line)] = struct{}{}
	}
	return out
}

func managedBlocks(doc document.Document) map[domain.TemplateID]managed.Block {
	out := map[domain.TemplateID]managed.Block{}
	for i, line := range doc.Lines {
		value := strings.TrimSpace(string(line.Text))
		if !strings.HasPrefix(value, "# >>> gitwatch:gitignore begin ") {
			continue
		}
		for j := i + 1; j < len(doc.Lines); j++ {
			end := strings.TrimSpace(string(doc.Lines[j].Text))
			if !strings.HasPrefix(end, "# <<< gitwatch:gitignore end ") {
				continue
			}
			raw := append([]byte(nil), doc.Bytes[doc.Lines[i].Start:doc.Lines[j].End]...)
			block, err := managed.ParseManagedBlock(raw)
			if err == nil {
				out[block.TemplateID] = block
			}
			break
		}
	}
	return out
}

func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
