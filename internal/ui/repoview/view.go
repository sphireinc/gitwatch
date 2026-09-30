package repoview

import (
	"fmt"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/registry"
	"github.com/sphireinc/git-watch/internal/ui/activityviz"
)

// Model stores filtered, sorted repository rows and the current selection.
type Model struct {
	Rows            []registry.Row
	AllRows         []registry.Row
	Selected        int
	Query           string
	Sort            registry.SortKey
	Desc            bool
	VisualsEnabled  bool
	ActivityBuckets int
}

// New creates a repository view model from registry rows.
func New(rows []registry.Row) Model {
	m := Model{AllRows: append([]registry.Row(nil), rows...), Sort: registry.SortName, VisualsEnabled: true, ActivityBuckets: 8}
	m.apply()
	return m
}

// SetVisualization controls the optional dense dashboard indicators. The
// bucket count is bounded here so configuration cannot make rendering scale
// with an unbounded history slice.
func (m *Model) SetVisualization(enabled bool, buckets int) {
	if buckets < 1 {
		buckets = 1
	}
	if buckets > 32 {
		buckets = 32
	}
	m.VisualsEnabled, m.ActivityBuckets = enabled, buckets
}

// SetRows replaces repository rows while preserving selection when possible.
func (m *Model) SetRows(rows []registry.Row) {
	selected := ""
	if m.Selected >= 0 && m.Selected < len(m.Rows) {
		selected = m.Rows[m.Selected].Repository.Path
	}
	m.AllRows = append([]registry.Row(nil), rows...)
	m.apply()
	m.Selected = 0
	for i, row := range m.Rows {
		if row.Repository.Path == selected {
			m.Selected = i
			break
		}
	}
}

// SetFilter applies a case-insensitive repository filter.
func (m *Model) SetFilter(query string) {
	m.Query = query
	m.apply()
	m.Selected = 0
}

// CycleSort advances through repository sort fields and direction.
func (m *Model) CycleSort() registry.SortKey {
	keys := []registry.SortKey{registry.SortName, registry.SortDirty, registry.SortAhead, registry.SortBehind}
	index := 0
	for i, key := range keys {
		if key == m.Sort {
			index = i
			break
		}
	}
	if index == len(keys)-1 {
		m.Desc = !m.Desc
		if !m.Desc {
			index = 0
		}
	} else {
		index++
	}
	m.Sort = keys[index]
	m.apply()
	m.Selected = 0
	return m.Sort
}

func (m *Model) apply() {
	filtered := registry.FilterRows(m.AllRows, m.Query)
	m.Rows = registry.SortRows(filtered, m.Sort, m.Desc)
}

// Move shifts the selected repository and clamps it to visible rows.
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

// VisibleWindow returns the row range that fits within maxContentLines while
// keeping the selected repository visible. A non-positive limit is unbounded.
func (m Model) VisibleWindow(maxContentLines int) (offset, count int) {
	total := len(m.Rows)
	if total == 0 {
		return 0, 0
	}
	if maxContentLines <= 0 {
		return 0, total
	}
	count = min(total, max(0, (maxContentLines-1)/2))
	if count == 0 {
		return 0, 0
	}
	selected := min(max(m.Selected, 0), total-1)
	if selected >= count {
		offset = selected - count + 1
	}
	if offset+count > total {
		offset = total - count
	}
	return offset, count
}

// View renders repositories and their current health state. An optional
// content-line limit keeps each repository's two-line presentation intact.
func (m Model) View(maxContentLines ...int) string {
	header := "Repositories"
	if m.Query != "" {
		header += " · filter: " + platform.SafeText(m.Query)
	}
	header += fmt.Sprintf(" · sort: %s", m.Sort)
	lines := []string{header}
	limit := 0
	if len(maxContentLines) > 0 {
		limit = maxContentLines[0]
	}
	offset, count := m.VisibleWindow(limit)
	end := offset + count
	now := time.Now()
	for i, row := range m.Rows[offset:end] {
		selectedIndex := offset + i
		prefix := "  "
		if selectedIndex == m.Selected {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%s · %s [%s] health:%s", prefix, platform.SafeText(row.Repository.Name), platform.SafeText(row.Branch), row.State, platform.SafeText(string(row.Health.Severity)))
		if row.Health.SigningEnabled {
			format := row.Health.SigningFormat
			if format == "" {
				format = "unset"
			}
			line += " signing:" + platform.SafeText(format)
		}
		if row.Operation != "" {
			line += " op:" + platform.SafeText(row.Operation)
		}
		if row.Attention != "" {
			line += " attention:" + platform.SafeText(row.Attention)
		}
		if row.RemoteFetchStatus != "" || !row.RemoteFetchAt.IsZero() {
			status := row.RemoteFetchStatus
			if status == "" {
				status = "unknown"
			}
			line += " remote-fetch:" + platform.SafeText(status)
			if row.RemoteFetchError != "" {
				line += "/" + platform.SafeText(row.RemoteFetchError)
			}
			if row.Repository.LastAutoFetchMillis > 0 {
				line += fmt.Sprintf(" latency:%dms", row.Repository.LastAutoFetchMillis)
			}
			line += " age:" + formatFetchAge(row.RemoteFetchAt, now)
		}
		line += fmt.Sprintf(" dirty:%d +%d/-%d stashes:%d worktrees:%d remotes:%d", row.Dirty, row.Ahead, row.Behind, row.Stashes, row.Worktrees, row.Remotes)
		if m.VisualsEnabled {
			heat := activityviz.HeatLevel(row.Staged, row.Unstaged, row.Untracked, row.Conflicts)
			activity := row.Activity
			if len(activity) == 0 {
				activity = []int{row.Ahead, row.Behind, row.Dirty, row.Conflicts}
			}
			line += fmt.Sprintf(" heat:%s%d diff:%s activity:%s", activityviz.HeatGlyph(heat), heat, activityviz.Bar(row.Staged+row.Unstaged, 10, 5), activityviz.Sparkline(activity, m.ActivityBuckets))
		}
		line += " gitignore:" + gitignoreLabel(row.Gitignore)
		if len(row.Warnings) > 0 {
			line += fmt.Sprintf(" warnings:%d", len(row.Warnings))
		}
		lines = append(lines, line)

		details := make([]string, 0, 3)
		if !row.Health.FreshAt.IsZero() {
			source := row.Health.Source
			if source == "" {
				source = "local"
			}
			details = append(details, "source:"+platform.SafeText(source)+" observed:"+row.Health.FreshAt.Format("15:04:05"))
		}
		if !row.RemoteFetchAt.IsZero() {
			details = append(details, "fetch@"+row.RemoteFetchAt.Format("15:04:05"))
		}
		if row.ProviderCIState != "" {
			freshness := "(fresh)"
			if row.ProviderCIStale {
				freshness = "(stale)"
			}
			provider := "ci:" + platform.SafeText(row.ProviderCIState) + freshness
			if row.ProviderCIAttention != "" {
				provider += "/" + platform.SafeText(row.ProviderCIAttention)
			}
			details = append(details, provider)
		}
		pathLine := "    " + platform.SafeText(row.Repository.Path)
		if len(details) > 0 {
			pathLine = "    " + strings.Join(details, " · ") + " · " + platform.SafeText(row.Repository.Path)
		}
		lines = append(lines, pathLine)
	}
	if len(m.Rows) == 0 {
		lines = append(lines, "  No repositories")
	}
	return strings.Join(lines, "\n")
}

func formatFetchAge(observed, now time.Time) string {
	if observed.IsZero() {
		return "unknown"
	}
	age := now.Sub(observed)
	if age < 0 {
		return "clock-skew"
	}
	switch {
	case age < time.Minute:
		return "just now"
	case age < time.Hour:
		return fmt.Sprintf("%dm ago", int(age/time.Minute))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(age/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(age/(24*time.Hour)))
	}
}

func gitignoreLabel(health registry.GitignoreHealth) string {
	if !health.Exists {
		return "absent"
	}
	return fmt.Sprintf("managed:%d partial:%d unmanaged:%d attention:%d updates:%d", health.Managed, health.Partial, health.Unmanaged, health.Attention, health.Updates)
}
