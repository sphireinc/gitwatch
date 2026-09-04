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

// New creates a history view from commits ordered by the history service.
func New(commits []history.Commit) Model { return Model{Rows: history.BuildGraph(commits)} }

// SetCommits replaces history rows and preserves the selected position.
func (m *Model) SetCommits(commits []history.Commit) {
	selectedSHA := ""
	if m.Selected >= 0 && m.Selected < len(m.Rows) {
		selectedSHA = m.Rows[m.Selected].Commit.SHA
	}
	m.Rows = history.BuildGraph(history.Filter(commits, m.Filter))
	m.Selected = 0
	for i, row := range m.Rows {
		if row.Commit.SHA == selectedSHA {
			m.Selected = i
			break
		}
	}
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
		lane := "│"
		if row.Lane == 0 {
			lane = "●"
		}
		if i == m.Selected && m.Pulse%2 == 1 {
			lane = "◉"
		}
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

func containsSHA(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
