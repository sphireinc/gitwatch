package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/operations"
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
	return layout.ComputeWithSplit(width, height, m.PanelSplit)
}

func (m Model) statusRowCount() int {
	statusLayout := m.statusLayout()
	width := statusLayout.Files.Width
	if statusLayout.Mode == layout.Wide {
		width = max(1, width-1)
	}
	width = max(1, width-1)
	rows := 0
	for _, height := range m.statusFileRowHeights(width) {
		if rows+height > max(1, statusLayout.Files.Height-1) {
			break
		}
		rows++
	}
	return max(1, rows)
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

	fileWidth := statusLayout.Files.Width
	if statusLayout.Mode == layout.Wide {
		fileWidth = max(1, fileWidth-1)
	}
	files := m.statusFileLines(fileWidth, statusLayout.Files.Height)
	diff := m.statusDiffLines(statusLayout.Details.Width, statusLayout.Details.Height)
	if statusLayout.Mode == layout.Wide {
		lines = append(lines, joinStatusColumns(files, diff, fileWidth, statusLayout.Details.Width, statusLayout.Files.Height)...)
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
	if m.OperationEngine != nil {
		active := 0
		for _, operation := range m.OperationEngine.Snapshot() {
			if operation.State == operations.Pending || operation.State == operations.Running {
				active++
			}
		}
		if active > 0 {
			suffix := "s"
			if active == 1 {
				suffix = ""
			}
			footer = fmt.Sprintf("[%d operation%s active] ", active, suffix) + footer
		}
	}
	if m.Notifications != nil && m.Notifications.Attention() > 0 {
		footer = fmt.Sprintf("[!] %d attention  [ctrl+n] dismiss  ", m.Notifications.Attention()) + footer
	}
	lines = append(lines, fitSafeDisplay(footer, width))
	return strings.Join(m.styleStatusLines(lines, statusLayout), "\n")
}

func (m Model) styleStatusLines(lines []string, statusLayout layout.Layout) []string {
	if len(lines) < 3 {
		return lines
	}
	styled := append([]string(nil), lines...)
	styled[0] = m.Theme.Header.Render(styled[0])
	styled[1] = m.styleStatusMetrics(styled[1])
	styled[2] = m.Theme.Border.Render(styled[2])

	contentStart := 3
	contentEnd := min(len(styled), contentStart+statusLayout.Files.Height)
	for index := contentStart; index < contentEnd; index++ {
		if statusLayout.Mode == layout.Wide {
			left, right, found := strings.Cut(styled[index], "│")
			if found {
				styled[index] = m.styleStatusFileLine(left) + m.Theme.Border.Render("│") + m.styleStatusDetailsLine(right, index == contentStart)
				continue
			}
		}
		if m.DiffPath != "" {
			styled[index] = m.styleStatusDetailsLine(styled[index], index == contentStart)
		} else {
			styled[index] = m.styleStatusFileLine(styled[index])
		}
	}

	if contentEnd < len(styled) {
		styled[contentEnd] = m.Theme.Border.Render(styled[contentEnd])
	}
	if len(styled) > 0 {
		styled[len(styled)-1] = m.Theme.Muted.Render(styled[len(styled)-1])
	}
	return styled
}

func (m Model) styleStatusMetrics(line string) string {
	parts := strings.Split(line, "  ")
	styles := []lipgloss.Style{m.Theme.Staged, m.Theme.Modified, m.Theme.Untracked, m.Theme.Conflict}
	for index := range parts {
		if index < len(styles) {
			parts[index] = styles[index].Render(parts[index])
		}
	}
	return strings.Join(parts, "  ")
}

func (m Model) styleStatusFileLine(line string) string {
	if strings.Contains(line, "clean worktree") {
		return m.Theme.Clean.Render(line)
	}
	if len(line) < 6 {
		return line
	}
	if line[3] == '>' {
		return m.Theme.Selection.Render(line)
	}
	role := m.Theme.Muted
	switch line[5] {
	case 'S':
		role = m.Theme.Staged
	case 'M', 'R':
		role = m.Theme.Modified
	case '?':
		role = m.Theme.Untracked
	case '!':
		role = m.Theme.Conflict
	case 'D':
		role = m.Theme.Deleted
	}
	styled := line[:5] + role.Render(line[5:6]) + line[6:]
	if line[:3] == "[S]" {
		styled = m.Theme.Staged.Render(line[:3]) + styled[3:]
	}
	return styled
}

func (m Model) styleStatusDetailsLine(line string, heading bool) string {
	if heading {
		return m.Theme.Header.Render(line)
	}
	trimmed := strings.TrimLeft(line, " ")
	switch {
	case strings.HasPrefix(trimmed, "@@"):
		return m.Theme.Header.Render(line)
	case strings.HasPrefix(trimmed, "+"):
		return m.Theme.Success.Render(line)
	case strings.HasPrefix(trimmed, "-"):
		return m.Theme.Deleted.Render(line)
	case strings.HasPrefix(trimmed, "Unable to load diff:"):
		return m.Theme.Error.Render(line)
	case strings.HasPrefix(trimmed, "Loading"):
		return m.Theme.Muted.Render(line)
	default:
		return line
	}
}

func (m Model) statusFileLines(width, height int) []string {
	lines := make([]string, 0, height)
	for i := m.Files.Offset; i < len(m.Files.Visible) && len(lines) < height; i++ {
		entry := m.Files.Entries[m.Files.Visible[i]]
		wrapped := fitSafeDisplayLines(m.statusFileText(entry, i == m.Files.Selected), width)
		lines = append(lines, wrapped...)
	}
	if len(lines) == 0 {
		lines = append(lines, fitSafeDisplay("  clean worktree", width))
	}
	return padStatusPanel(lines, width, height)
}

func (m Model) statusFileRowHeights(width int) []int {
	heights := make([]int, 0, len(m.Files.Visible))
	for i := m.Files.Offset; i < len(m.Files.Visible); i++ {
		entry := m.Files.Entries[m.Files.Visible[i]]
		heights = append(heights, len(fitSafeDisplayLines(m.statusFileText(entry, i == m.Files.Selected), width)))
	}
	return heights
}

func (m Model) statusFileText(entry repo.Entry, selected bool) string {
	stage := "[ ]"
	if entry.Staged {
		stage = "[S]"
	}
	selection := " "
	if selected {
		selection = ">"
	}
	path := string(entry.Path)
	if len(entry.Original) > 0 {
		path = string(entry.Original) + " → " + path
	}
	return fmt.Sprintf("%s%s %s %s", stage, selection, theme.Symbol(repo.StatusLabel(entry)), path)
}

func (m Model) statusDiffLines(width, height int) []string {
	if m.DiffPath == "" {
		return padStatusPanel(m.statusDetailsLines(width, height), width, height)
	}
	mode := "unstaged"
	if m.DiffStaged {
		mode = "staged"
	}
	heading := fmt.Sprintf("Diff (%s) · %s · +%d -%d · [V] switch · [/] search · [esc] close", mode, m.DiffPath, m.DiffAdded, m.DiffDeleted)
	if m.DiffSearchInput != "" {
		heading += " · search: " + m.DiffSearchInput
	}
	lines := []string{fitSafeDisplay(heading, width)}
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
	if m.DiffTruncated {
		lines = append([]string{"diff truncated at configured budget"}, lines...)
	}
	return padStatusPanel(lines, width, height)
}

// padStatusPanel reserves one terminal cell on the panel's left and top edges.
// A cell is the smallest reliable spacing unit across terminal fonts and keeps
// the visual inset aligned with mouse coordinates and wrapped content.
func padStatusPanel(lines []string, width, height int) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	innerWidth := max(1, width-1)
	padded := make([]string, 0, len(lines)+1)
	padded = append(padded, "")
	for _, line := range lines {
		wrapped := fitSafeDisplayLines(line, innerWidth)
		for _, part := range wrapped {
			padded = append(padded, " "+part)
		}
	}
	return fitLines(padded, width, height)
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
	joined := make([]string, height)
	for index := range joined {
		var leftLine, rightLine string
		if index < len(left) {
			leftLine = left[index]
		}
		if index < len(right) {
			rightLine = right[index]
		}
		joined[index] = fitDisplay(leftLine, leftWidth) + "│" + fitDisplay(rightLine, rightWidth)
	}
	return joined
}

func fitLines(lines []string, width, height int) []string {
	fitted := make([]string, height)
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, fitSafeDisplayLines(line, width)...)
	}
	for index := range fitted {
		if index < len(wrapped) {
			fitted[index] = wrapped[index]
		} else {
			fitted[index] = strings.Repeat(" ", max(0, width))
		}
	}
	return fitted
}

func fitSafeDisplay(value string, width int) string {
	return fitDisplay(platform.SafeText(platform.RedactSecrets(value)), width)
}

func fitSafeDisplayLines(value string, width int) []string {
	return wrapDisplay(platform.SafeText(platform.RedactSecrets(value)), width)
}

func wrapDisplay(value string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	value = strings.ReplaceAll(value, "\t", "    ")
	if value == "" {
		return []string{""}
	}
	lines := make([]string, 0, 1+(runewidth.StringWidth(value)/width))
	for len(value) > 0 {
		line, remainder := takeDisplayWidth(value, width)
		lines = append(lines, line)
		value = remainder
	}
	return lines
}

func takeDisplayWidth(value string, width int) (string, string) {
	used := 0
	cut := 0
	for index, r := range value {
		runeWidth := runewidth.RuneWidth(r)
		if used > 0 && used+runeWidth > width {
			break
		}
		if used == 0 && runeWidth > width {
			cut = index + len(string(r))
			break
		}
		used += runeWidth
		cut = index + len(string(r))
		if used >= width {
			break
		}
	}
	if cut == 0 {
		return value, ""
	}
	return value[:cut], value[cut:]
}

func fitDisplay(value string, width int) string {
	if width <= 0 {
		return ""
	}
	value = strings.ReplaceAll(value, "\t", "    ")
	value = runewidth.Truncate(value, width, "")
	return runewidth.FillRight(value, width)
}
