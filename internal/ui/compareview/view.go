// Package compareview renders a bounded arbitrary-revision comparison.
package compareview

import (
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/compare"
	"github.com/sphireinc/git-watch/internal/platform"
)

type Model struct {
	Result         compare.Result
	Selected       int
	SelectedPatch  string
	PatchPath      string
	PatchLoading   bool
	PatchTruncated bool
}

func (m *Model) SetResult(result compare.Result) { m.Result, m.Selected = result, 0 }

func (m *Model) SetPatch(path, text string, truncated bool) {
	m.PatchPath, m.SelectedPatch, m.PatchTruncated, m.PatchLoading = path, text, truncated, false
}

func (m *Model) Move(delta int) {
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Result.Changes) {
		m.Selected = max(0, len(m.Result.Changes)-1)
	}
}

func (m Model) View() string {
	lines := []string{
		"Comparison",
		"A: " + platform.SafeText(m.Result.Left.Ref) + " (" + platform.SafeText(m.Result.Left.SHA) + ")",
		"B: " + platform.SafeText(m.Result.Right.Ref) + " (" + platform.SafeText(m.Result.Right.SHA) + ")",
		fmt.Sprintf("Changed files: %d", len(m.Result.Changes)),
	}
	if m.Result.FilesTruncated {
		lines = append(lines, "NOTICE: changed-file list truncated by budget")
	}
	lines = append(lines, "", "Changed paths:")
	for index, change := range m.Result.Changes {
		prefix := "  "
		if index == m.Selected {
			prefix = "> "
		}
		path := change.NewPath
		if strings.HasPrefix(change.Status, "R") || strings.HasPrefix(change.Status, "C") {
			path = change.OldPath + " -> " + change.NewPath
		}
		stats := ""
		if change.Binary {
			stats = " binary"
		} else {
			stats = fmt.Sprintf(" +%d -%d", change.Added, change.Removed)
		}
		lines = append(lines, prefix+platform.SafeText(change.Status)+" "+platform.SafeText(path)+stats)
	}
	if len(m.Result.Changes) == 0 {
		lines = append(lines, "  No changed files")
	}
	patch := m.Result.Patch
	patchLabel := "Patch:"
	if m.PatchPath != "" {
		patch, patchLabel = m.SelectedPatch, "Selected patch ("+platform.SafeText(m.PatchPath)+"):"
	}
	if m.PatchLoading {
		lines = append(lines, "", "Loading selected patch…")
	} else if patch != "" {
		lines = append(lines, "", patchLabel, platform.SafeText(patch))
	}
	if m.Result.PatchTruncated || m.PatchTruncated {
		lines = append(lines, "Patch truncated by budget")
	}
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
