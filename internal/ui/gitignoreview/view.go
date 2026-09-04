// Package gitignoreview provides the repository-scoped, keyboard-first catalog browser.
package gitignoreview

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/match"
)

type Tab string

const (
	All         Tab = "all"
	Common      Tab = "common"
	Global      Tab = "global"
	Community   Tab = "community"
	Installed   Tab = "installed"
	Recommended Tab = "recommended"
)

// Entry is the immutable display projection of one catalog template.
type Entry struct {
	Template    catalog.Template
	Match       match.Result
	Selected    bool
	Recommended bool
}

// RepositoryModel contains no process or filesystem state. RepositoryID keeps
// selections from different repositories independent when callers host more
// than one browser at once.
type RepositoryModel struct {
	RepositoryID   domain.RepositoryID
	CatalogVersion string
	AllEntries     []Entry
	Entries        []Entry
	Query          string
	Tab            Tab
	Selected       int
	DetailsOffset  int
	Width, Height  int
}

func New(repoID domain.RepositoryID, cat *catalog.Catalog, results []match.Result) RepositoryModel {
	m := RepositoryModel{RepositoryID: repoID, Tab: All, Width: 80, Height: 24}
	if cat == nil {
		return m
	}
	byID := make(map[domain.TemplateID]match.Result, len(results))
	for _, result := range results {
		byID[result.TemplateID] = result
	}
	for _, template := range cat.List() {
		result := byID[template.ID]
		m.AllEntries = append(m.AllEntries, Entry{Template: template, Match: result, Recommended: result.Kind == domain.Partial})
	}
	m.CatalogVersion = cat.Version()
	m.apply()
	return m
}

func (m *RepositoryModel) SetMatches(results []match.Result) {
	byID := make(map[domain.TemplateID]match.Result, len(results))
	for _, result := range results {
		byID[result.TemplateID] = result
	}
	for i := range m.AllEntries {
		if result, ok := byID[m.AllEntries[i].Template.ID]; ok {
			m.AllEntries[i].Match = result
			m.AllEntries[i].Recommended = result.Kind == domain.Partial
		}
	}
	m.apply()
}

func (m *RepositoryModel) SetQuery(query string) { m.Query = query; m.apply(); m.Selected = 0 }
func (m *RepositoryModel) SetTab(tab Tab)        { m.Tab = tab; m.apply(); m.Selected = 0 }
func (m *RepositoryModel) SetSize(width, height int) {
	if width > 0 {
		m.Width = width
	}
	if height > 0 {
		m.Height = height
	}
}

func (m *RepositoryModel) apply() {
	selected := make(map[domain.TemplateID]bool, len(m.AllEntries))
	for _, e := range m.AllEntries {
		selected[e.Template.ID] = e.Selected
	}
	query := normalize(m.Query)
	m.Entries = m.Entries[:0]
	for i := range m.AllEntries {
		e := m.AllEntries[i]
		e.Selected = selected[e.Template.ID]
		if !tabMatch(m.Tab, e) || (query != "" && fuzzyScore(e, query) < 0) {
			continue
		}
		m.Entries = append(m.Entries, e)
	}
	sort.SliceStable(m.Entries, func(i, j int) bool {
		a, b := m.Entries[i], m.Entries[j]
		if priority(a) != priority(b) {
			return priority(a) < priority(b)
		}
		if m.Query != "" && fuzzyScore(a, query) != fuzzyScore(b, query) {
			return fuzzyScore(a, query) > fuzzyScore(b, query)
		}
		return strings.ToLower(a.Template.ID.String()) < strings.ToLower(b.Template.ID.String())
	})
}

func priority(e Entry) int {
	if e.Match.Kind.Full() {
		return 0
	}
	if e.Recommended {
		return 1
	}
	return 2
}

func tabMatch(tab Tab, e Entry) bool {
	switch tab {
	case Common:
		return e.Template.Category == domain.CategoryRoot
	case Global:
		return e.Template.Category == domain.CategoryGlobal
	case Community:
		return e.Template.Category == domain.CategoryCommunity
	case Installed:
		return e.Match.Kind.Full() || e.Match.Kind == domain.ManagedEdited
	case Recommended:
		return e.Recommended
	default:
		return true
	}
}

func fuzzyScore(e Entry, q string) int {
	values := []string{e.Template.Name, e.Template.ID.String(), string(e.Template.Category), e.Template.SourcePath}
	for _, alias := range e.Template.Aliases {
		values = append(values, alias)
	}
	best := -1
	for _, value := range values {
		if score := subsequence(normalize(value), q); score > best {
			best = score
		}
	}
	return best
}

func subsequence(value, query string) int {
	if query == "" {
		return 0
	}
	pos, score := 0, 0
	for _, needle := range query {
		found := -1
		for pos < len([]rune(value)) {
			if []rune(value)[pos] == needle {
				found = pos
				pos++
				break
			}
			pos++
		}
		if found < 0 {
			return -1
		}
		score += 100 - found
	}
	return score
}

func normalize(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (m *RepositoryModel) Move(delta int) {
	if len(m.Entries) == 0 {
		m.Selected = 0
		return
	}
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Entries) {
		m.Selected = len(m.Entries) - 1
	}
}

func (m *RepositoryModel) Toggle() bool {
	if m.Selected < 0 || m.Selected >= len(m.Entries) {
		return false
	}
	id := m.Entries[m.Selected].Template.ID
	for i := range m.AllEntries {
		if m.AllEntries[i].Template.ID == id {
			m.AllEntries[i].Selected = !m.AllEntries[i].Selected
			break
		}
	}
	m.apply()
	return true
}
func (m *RepositoryModel) ClearSelection() {
	for i := range m.AllEntries {
		m.AllEntries[i].Selected = false
	}
	m.apply()
}
func (m RepositoryModel) SelectedEntries() []Entry {
	out := []Entry{}
	for _, e := range m.AllEntries {
		if e.Selected {
			out = append(out, e)
		}
	}
	return out
}

// UpdateKey returns whether the key was consumed. Input is deliberately a
// string so the model is simple to test and can be adapted to Bubble Tea.
func (m *RepositoryModel) UpdateKey(key string) (consumed bool) {
	switch key {
	case "up", "k":
		m.Move(-1)
	case "down", "j":
		m.Move(1)
	case "space":
		m.Toggle()
	case "backspace":
		if r := []rune(m.Query); len(r) > 0 {
			m.SetQuery(string(r[:len(r)-1]))
		}
	case "tab":
		m.SetTab(nextTab(m.Tab))
	case "esc":
		return true
	default:
		if len([]rune(key)) == 1 {
			m.SetQuery(m.Query + key)
		} else {
			return false
		}
	}
	return true
}

func nextTab(tab Tab) Tab {
	tabs := []Tab{All, Common, Global, Community, Installed, Recommended}
	for i, v := range tabs {
		if v == tab {
			return tabs[(i+1)%len(tabs)]
		}
	}
	return All
}

func indicator(e Entry) string {
	if e.Selected {
		if e.Match.Kind.Full() {
			return "-"
		}
		return "+"
	}
	switch e.Match.Kind {
	case domain.ManagedEdited, domain.InvalidManagedBlock:
		return "!"
	case domain.Partial:
		return "~"
	case domain.ManagedExact, domain.UnmanagedFull:
		return "*"
	default:
		return " "
	}
}

func (m *RepositoryModel) Click(x, y int) bool {
	if y < 4 || y >= 4+len(m.Entries) {
		return false
	}
	index := y - 4
	if index >= 0 && index < len(m.Entries) {
		m.Selected = index
		if x <= 3 {
			m.Toggle()
		}
		return true
	}
	return false
}

func (m RepositoryModel) View() string {
	lines := []string{fmt.Sprintf("Gitignore catalog · repository %s", m.RepositoryID), fmt.Sprintf("Search: %s · %d results · tab: %s · catalog: %s", m.Query, len(m.Entries), m.Tab, short(m.CatalogVersion)), "", "Tabs: [all] common global community installed recommended"}
	for i, e := range m.Entries {
		prefix := "  "
		if i == m.Selected {
			prefix = "> "
		}
		line := fmt.Sprintf("%s[%s] %-24s %-9s %d/%d", prefix, indicator(e), e.Template.Name, e.Template.Category, e.Match.Present, e.Match.Total)
		if i == m.Selected {
			line += "  " + e.Template.SourcePath
		}
		lines = append(lines, line)
	}
	if len(m.Entries) == 0 {
		lines = append(lines, "  No matching templates")
	}
	if len(m.Entries) > 0 {
		e := m.Entries[m.Selected]
		lines = append(lines, "", "Details: "+e.Template.Name, "source: "+e.Template.SourcePath, "state: "+string(e.Match.Kind), "preview:")
		preview := strings.Split(string(e.Template.Content), "\n")
		limit := 5
		if m.Height > 24 {
			limit = 10
		}
		for i := 0; i < len(preview) && i < limit; i++ {
			lines = append(lines, "  "+preview[i])
		}
	}
	lines = append(lines, "", "[j/k] move [space] select [/] search [tab] filter [a] add selected [d] remove [p] preview [r] refresh [c] clear [esc] back")
	return strings.Join(lines, "\n")
}

func short(v string) string {
	if len(v) > 8 {
		return v[:8]
	}
	return v
}
