// Package historyview renders commit history and graph lanes.
package historyview

import (
	"fmt"
	"strings"

	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/platform"
)

// Model owns the small amount of interaction state needed to render history.
// Git loading is deliberately kept outside this package so views remain pure.
type Model struct {
	Rows     []history.GraphRow
	Selected int
	Filter   string
	Pulse    uint8
	Basket   history.Selection
	ASCII    bool
	commits  []history.Commit
	cursor   history.GraphCursor
}

// SetScope binds the basket to the current repository and ref. A scope switch
// clears prior selection so SHAs cannot leak into another repository.
func (m *Model) SetScope(repository, ref string, generation uint64) error {
	if m.Basket.Repository() == repository && m.Basket.Ref() == ref {
		return nil
	}
	basket, err := history.NewSelection(repository, ref, generation)
	if err != nil {
		return err
	}
	m.Basket = basket
	return nil
}

// ToggleBasket toggles the selected history row in the scoped basket.
func (m *Model) ToggleBasket() error {
	if m.Selected < 0 || m.Selected >= len(m.Rows) {
		return nil
	}
	basket, err := m.Basket.Toggle(m.Rows[m.Selected].Commit.SHA)
	if err == nil {
		m.Basket = basket
	}
	return err
}

// ClearBasket clears selected commits without changing scope.
func (m *Model) ClearBasket() { m.Basket = m.Basket.Clear() }

// SetASCII selects the portable graph glyphs used by terminals that cannot
// reliably render the Unicode lane markers. It does not alter graph state.
func (m *Model) SetASCII(ascii bool) { m.ASCII = ascii }

// SelectRange adds the inclusive visible-row range to the scoped basket. Rows
// are displayed newest-first, while Selection normalizes the basket to Git's
// oldest-first application order.
func (m *Model) SelectRange(start, end int) error {
	commits := make([]history.Commit, len(m.Rows))
	for index, row := range m.Rows {
		commits[index] = row.Commit
	}
	basket, err := m.Basket.SelectRange(commits, start, end)
	if err != nil {
		return err
	}
	m.Basket = basket
	return nil
}

// New creates a history view from commits ordered by the history service.
func New(commits []history.Commit) Model {
	var model Model
	model.SetCommits(commits)
	return model
}

// SetCommits replaces history rows and preserves the selected position.
func (m *Model) SetCommits(commits []history.Commit) {
	selectedSHA := ""
	if m.Selected >= 0 && m.Selected < len(m.Rows) {
		selectedSHA = m.Rows[m.Selected].Commit.SHA
	}
	m.commits = append([]history.Commit(nil), commits...)
	m.Rows = history.BuildGraph(history.Filter(m.commits, m.Filter))
	_, m.cursor = history.BuildGraphPage(m.commits, history.GraphCursor{})
	m.Selected = 0
	for i, row := range m.Rows {
		if row.Commit.SHA == selectedSHA {
			m.Selected = i
			break
		}
	}
}

// AppendCommits adds the next unfiltered history page without recomputing
// existing lanes. This preserves merge topology at the page boundary.
func (m *Model) AppendCommits(commits []history.Commit) {
	if len(commits) == 0 {
		return
	}
	m.commits = append(m.commits, commits...)
	if m.Filter != "" {
		m.SetCommits(m.commits)
		return
	}
	rows, cursor := history.BuildGraphPage(commits, m.cursor)
	m.Rows = append(m.Rows, rows...)
	m.cursor = cursor
}

// SetFilter applies a subject, author, or SHA filter to commits.
func (m *Model) SetFilter(filter string, commits []history.Commit) {
	m.Filter = filter
	m.SetCommits(commits)
}

// SetPulse updates the render pulse used for activity emphasis.
func (m *Model) SetPulse(pulse uint8) { m.Pulse = pulse }

// Move shifts the selected commit and clamps it to available rows.
func (m *Model) Move(delta int) {
	if len(m.Rows) == 0 {
		m.Selected = 0
		return
	}
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Rows) {
		m.Selected = len(m.Rows) - 1
	}
}

// View renders the history graph and selected commit metadata.
func (m Model) View() string {
	lines := []string{"History"}
	if m.Basket.Count() > 0 {
		lines = append(lines, fmt.Sprintf("Basket: %d commit(s), application order oldest first", m.Basket.Count()))
		lines = append(lines, "Revert order preview:")
		shas := m.Basket.SHAs()
		previewCount := len(shas)
		if previewCount > 32 {
			previewCount = 32
		}
		for index := 0; index < previewCount; index++ {
			lines = append(lines, fmt.Sprintf("  %d. %s", index+1, platform.SafeText(shas[index])))
		}
		if previewCount < len(shas) {
			lines = append(lines, fmt.Sprintf("  … %d more commit(s)", len(shas)-previewCount))
		}
	}
	if m.Filter != "" {
		lines = append(lines, "Filter: "+m.Filter)
	}
	for i, row := range m.Rows {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		if containsSHA(m.Basket.SHAs(), row.Commit.SHA) {
			prefix = "* "
		}
		lane := graphLanePrefix(row, i == m.Selected && m.Pulse%2 == 1, m.ASCII)
		refs := append([]string{}, row.Branches...)
		refs = append(refs, row.Tags...)
		for j := range refs {
			refs[j] = platform.SafeText(refs[j])
		}
		decoration := ""
		if row.Head {
			decoration = " HEAD"
		}
		if len(refs) > 0 {
			decoration += " [" + strings.Join(refs, ", ") + "]"
		}
		lines = append(lines, fmt.Sprintf("%s%s %s %s%s", prefix, lane, platform.SafeText(row.Commit.Short), platform.SafeText(row.Commit.Subject), decoration))
	}
	if len(m.Rows) == 0 {
		lines = append(lines, "  No commits")
	}
	return strings.Join(lines, "\n")
}

func graphLanePrefix(row history.GraphRow, pulse, ascii bool) string {
	lanes := row.Lanes
	if lanes < 1 {
		lanes = 1
	}
	if row.Lane < 0 || row.Lane >= lanes {
		row.Lane = 0
	}
	columns := make([]rune, lanes)
	branch, commit, pulseCommit := '│', '●', '◉'
	if ascii {
		branch, commit, pulseCommit = '|', 'o', '@'
	}
	for index := range columns {
		columns[index] = branch
	}
	columns[row.Lane] = commit
	if pulse {
		columns[row.Lane] = pulseCommit
	}
	return string(columns)
}

func containsSHA(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
