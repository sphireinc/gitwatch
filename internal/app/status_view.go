package app

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/repo"
	"github.com/sphireinc/git-watch/internal/ui/details"
	"github.com/sphireinc/git-watch/internal/ui/layout"
	"github.com/sphireinc/git-watch/internal/ui/theme"
)

const (
	defaultStatusWidth  = 100
	defaultStatusHeight = 24
)

func (m Model) statusLayout() layout.Layout {
	width, height := m.Width, m.Height
	if width <= 0 {
		width = defaultStatusWidth
	}
	if height <= 0 {
		height = defaultStatusHeight
	}
	return layout.Compute(width, height)
}

func (m Model) statusRowCount() int {
	return max(1, m.statusLayout().Files.Height)
}

func (m *Model) scrollDiff(delta int) {
	if m.DiffPath == "" || m.DiffLoading || m.DiffBinary {
		return
	}
	lines := strings.Split(m.DiffText, "\n")
	viewport := max(1, m.statusRowCount()-1)
	maximum := max(0, len(lines)-viewport)
	m.DiffOffset += delta
	if m.DiffOffset < 0 {
		m.DiffOffset = 0
	}
	if m.DiffOffset > maximum {
		m.DiffOffset = maximum
	}
}

func (m Model) statusView() string {
	statusLayout := m.statusLayout()
	width := statusLayout.Header.Width
	if statusLayout.Mode == layout.TooSmall {
		return strings.Join([]string{
			fitDisplay("gitwatch", width),
			fitDisplay(statusLayout.Message, width),
			fitDisplay("[q] quit", width),
		}, "\n")
	}
	name := m.Snapshot.Branch.Name
	if name == "" {
		name = "repository"
	}
	watchLabel := watchModeName(m.WatchMode)
	if m.WatchMode != "" && m.Motion.Ticks() {
		watchLabel += map[bool]string{true: " ●", false: " ○"}[m.WatchPulse%2 == 0]
	}
	header := fmt.Sprintf("gitwatch · %s · %s · watch:%s", name, stateName(m.State), watchLabel)
	metrics := fmt.Sprintf("STAGED %d  MODIFIED %d  UNTRACKED %d  CONFLICTS %d", m.Snapshot.Counts.Staged, m.Snapshot.Counts.Unstaged, m.Snapshot.Counts.Untracked, m.Snapshot.Counts.Conflicted)
	lines := []string{fitSafeDisplay(header, width), fitSafeDisplay(metrics, width), strings.Repeat("─", width)}

	files := m.statusFileLines(statusLayout.Files.Width, statusLayout.Files.Height)
	diff := m.statusDiffLines(statusLayout.Details.Width, statusLayout.Details.Height)
	if statusLayout.Mode == layout.Wide {
		lines = append(lines, joinStatusColumns(files, diff, statusLayout.Files.Width, statusLayout.Details.Width, statusLayout.Files.Height)...)
	} else if m.DiffPath != "" {
		lines = append(lines, fitLines(diff, width, statusLayout.Files.Height)...)
	} else {
		lines = append(lines, fitLines(files, width, statusLayout.Files.Height)...)
	}

	lines = append(lines, strings.Repeat("─", width))
	lines = append(lines, fitSafeDisplay(m.Status, width))
	notice := m.latestActivityLine()
	if m.Toast.Text != "" {
		notice = "NOTICE: " + m.Toast.Text
	}
	lines = append(lines, fitSafeDisplay(notice, width))
	footer := "[j/k] move  [space] stage  [a/U] all  [enter/d] diff  [/] filter  [S] sort  [R] restore  [?] help  [q] quit"
	if m.Notifications != nil && m.Notifications.Attention() > 0 {
		footer = fmt.Sprintf("[!] %d attention  [ctrl+n] dismiss  ", m.Notifications.Attention()) + footer
	}
	lines = append(lines, fitSafeDisplay(footer, width))
	return strings.Join(lines, "\n")
}

func (m Model) statusFileLines(width, height int) []string {
	lines := make([]string, 0, height)
	for i := m.Files.Offset; i < len(m.Files.Visible) && len(lines) < height; i++ {
		entry := m.Files.Entries[m.Files.Visible[i]]
		stage := "[ ]"
		if entry.Staged {
			stage = "[S]"
		}
		selection := " "
		if i == m.Files.Selected {
			selection = ">"
		}
		path := string(entry.Path)
		if len(entry.Original) > 0 {
			path = string(entry.Original) + " → " + path
		}
		lines = append(lines, fitSafeDisplay(fmt.Sprintf("%s%s %s %s", stage, selection, theme.Symbol(repo.StatusLabel(entry)), path), width))
	}
	if len(lines) == 0 {
		lines = append(lines, fitDisplay("  clean worktree", width))
	}
	return fitLines(lines, width, height)
}

func (m Model) statusDiffLines(width, height int) []string {
	if m.DiffPath == "" {
		return m.statusDetailsLines(width, height)
	}
	mode := "unstaged"
	if m.DiffStaged {
		mode = "staged"
	}
	lines := []string{fitSafeDisplay(fmt.Sprintf("Diff (%s) · %s · +%d -%d · [esc] close", mode, m.DiffPath, m.DiffAdded, m.DiffDeleted), width)}
	switch {
	case m.DiffLoading:
		lines = append(lines, "", "Loading…")
	case m.DiffErr != nil:
		lines = append(lines, "", "Unable to load diff: "+m.DiffErr.Error())
	case m.DiffBinary:
		lines = append(lines, "", "[binary file — textual diff not rendered]")
	case m.DiffText == "":
		lines = append(lines, "", "No "+mode+" changes for this path.")
	default:
		body := strings.Split(m.DiffText, "\n")
		if m.DiffOffset < len(body) {
			body = body[m.DiffOffset:]
		} else {
			body = nil
		}
		lines = append(lines, body...)
	}
	return fitLines(lines, width, height)
}

func (m Model) statusDetailsLines(width, height int) []string {
	lines := []string{"Selected file details", "", "Select a file and press Enter/d, or click its row, to open its diff."}
	if m.Files.Selected < 0 || m.Files.Selected >= len(m.Files.Visible) {
		return fitLines(lines, width, height)
	}
	entry := m.Files.Entries[m.Files.Visible[m.Files.Selected]]
	var detail details.View
	if m.DetailsCache != nil {
		detail = m.DetailsCache.For(m.Snapshot, entry)
	} else {
		detail = details.Build(m.Snapshot, entry)
	}
	lines = []string{
		"Selected file details",
		"",
		"Path: " + detail.Path,
		"Status: " + detail.Status,
		"Index staged: " + fmt.Sprint(detail.Staged),
		"Worktree modified: " + fmt.Sprint(detail.Unstaged),
		"Mode: " + detail.Mode,
	}
	if detail.PreviousPath != "" {
		lines = append(lines, "Previous path: "+detail.PreviousPath)
	}
	if detail.Conflict {
		lines = append(lines, "Conflict: "+entry.ConflictType())
	}
	if detail.Submodule {
		lines = append(lines, "Submodule: "+detail.SubmoduleState)
	}
	lines = append(lines, "Diffstat: calculated when the diff is opened")
	if !detail.ObservedAt.IsZero() {
		lines = append(lines, "Observed: "+detail.ObservedAt.Format("15:04:05"))
	}
	lines = append(lines, "", detail.Hint)
	return fitLines(lines, width, height)
}

func (m Model) latestActivityLine() string {
	if m.ActivityLog == nil {
		return ""
	}
	events := m.ActivityLog.All()
	if len(events) == 0 {
		return ""
	}
	event := events[len(events)-1]
	text := "activity: " + string(event.Kind)
	if event.Path != "" {
		text += " · " + event.Path
	}
	if event.Message != "" {
		text += " · " + event.Message
	}
	return text
}

func joinStatusColumns(left, right []string, leftWidth, rightWidth, height int) []string {
	leftContentWidth := max(1, leftWidth-1)
	joined := make([]string, height)
	for index := range joined {
		var leftLine, rightLine string
		if index < len(left) {
			leftLine = left[index]
		}
		if index < len(right) {
			rightLine = right[index]
		}
		joined[index] = fitDisplay(leftLine, leftContentWidth) + "│" + fitDisplay(rightLine, rightWidth)
	}
	return joined
}

func fitLines(lines []string, width, height int) []string {
	fitted := make([]string, height)
	for index := range fitted {
		if index < len(lines) {
			fitted[index] = fitSafeDisplay(lines[index], width)
		} else {
			fitted[index] = strings.Repeat(" ", max(0, width))
		}
	}
	return fitted
}

func fitSafeDisplay(value string, width int) string {
	return fitDisplay(platform.SafeText(platform.RedactSecrets(value)), width)
}

func fitDisplay(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.ReplaceAll(value, "\t", "    ")
	value = runewidth.Truncate(value, width, "")
	return runewidth.FillRight(value, width)
}
