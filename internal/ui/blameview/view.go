// Package blameview renders a bounded, selectable blame page.
package blameview

import (
	"fmt"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/blame"
	"github.com/sphireinc/git-watch/internal/platform"
)

type Model struct {
	Path     string
	Lines    []blame.Line
	Start    int
	Selected int
	HasMore  bool
}

func (m *Model) SetPage(path string, start int, lines []blame.Line, hasMore bool) {
	m.Path, m.Start, m.HasMore = path, start, hasMore
	m.Lines = cloneLines(lines)
	m.Selected = 0
}

func (m *Model) AppendPage(start int, lines []blame.Line, hasMore bool) {
	if m.Start+len(m.Lines) != start {
		return
	}
	m.Lines = append(m.Lines, cloneLines(lines)...)
	m.HasMore = hasMore
}

func (m *Model) Move(delta int) {
	if len(m.Lines) == 0 {
		m.Selected = -1
		return
	}
	m.Selected += delta
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Selected >= len(m.Lines) {
		m.Selected = len(m.Lines) - 1
	}
}

func (m Model) SelectedLine() (blame.Line, bool) {
	if m.Selected < 0 || m.Selected >= len(m.Lines) {
		return blame.Line{}, false
	}
	return m.Lines[m.Selected], true
}

func (m Model) View() string {
	lines := []string{"Blame · " + clean(m.Path)}
	for index, line := range m.Lines {
		prefix := "  "
		if index == m.Selected {
			prefix = "> "
		}
		sha := line.FinalSHA
		if len(sha) > 10 {
			sha = sha[:10]
		}
		author := line.Author
		if author == "" {
			author = line.AuthorMail
		}
		when := ""
		if line.AuthorTime != 0 {
			when = time.Unix(line.AuthorTime, 0).Format("2006-01-02")
		}
		lines = append(lines, fmt.Sprintf("%s%6d %s · %-16s · %s · %s", prefix, m.Start+index, clean(sha), clean(author), when, clean(string(line.Content))))
	}
	if len(m.Lines) == 0 {
		lines = append(lines, "  No blame lines")
	}
	if m.HasMore {
		lines = append(lines, "", "[right bracket] load more lines")
	}
	return strings.Join(lines, "\n")
}

func cloneLines(lines []blame.Line) []blame.Line {
	cloned := make([]blame.Line, len(lines))
	for i, line := range lines {
		cloned[i] = line
		cloned[i].Content = append([]byte(nil), line.Content...)
	}
	return cloned
}

func clean(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
	return platform.SafeText(value)
}
