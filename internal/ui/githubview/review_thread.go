package githubview

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/provider"
)

func (m Model) reviewThreadLines(width int) []string {
	if len(m.Comments) == 0 {
		return []string{"No review comments loaded"}
	}
	selected := m.Comments[min(max(0, m.SelectedComment), len(m.Comments)-1)]
	lines := []string{fmt.Sprintf("Review thread #%d · [ / ] select · PgUp/PgDn scroll · j/k workflows", selected.ID)}
	if selected.InReplyTo > 0 {
		found := false
		for _, comment := range m.Comments {
			if comment.ID == selected.InReplyTo {
				lines = append(lines, reviewCommentLines(comment, width, "Parent")...)
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, fmt.Sprintf("Reply to #%d (parent not loaded)", selected.InReplyTo))
		}
	}
	return append(lines, reviewCommentLines(selected, width, "Selected")...)
}

func reviewCommentLines(comment provider.ReviewComment, width int, label string) []string {
	location := workflowText(comment.Path)
	if comment.Line > 0 {
		location += fmt.Sprintf(":%d", comment.Line)
	}
	lines := []string{fmt.Sprintf("%s #%d · @%s · %s", label, comment.ID, workflowText(comment.Author), location)}
	body := strings.ReplaceAll(platform.SafeText(comment.Body), "\t", "    ")
	lines = append(lines, strings.Split(runewidth.Wrap(body, max(1, width)), "\n")...)
	return append(lines, "")
}

func (m Model) reviewViewportLines(width, height int) []string {
	cacheState := "fresh"
	if m.ProviderStale {
		cacheState = "stale"
	}
	lines := []string{"GitHub · review thread", "Repository: " + workflowText(m.Repository.Owner) + "/" + workflowText(m.Repository.Name), "Branch: " + workflowText(m.Branch), "Provider cache: " + cacheState, "[Prev comment] [Next comment]"}
	content := m.reviewThreadLines(width)
	available := max(0, height-len(lines))
	offset := min(max(0, m.ReviewOffset), max(0, len(content)-available))
	return append(lines, content[offset:min(len(content), offset+available)]...)
}

// ScrollReview clamps the offset to the selected thread's already-loaded data.
func (m *Model) ScrollReview(delta, width, height int) {
	if !m.ReviewFocused {
		return
	}
	content := m.reviewThreadLines(max(1, width))
	m.ReviewOffset = min(max(0, m.ReviewOffset+delta), max(0, len(content)-max(0, height-5)))
}

func (m *Model) SelectReviewControl(row, x int) bool {
	if !m.ReviewFocused || !m.Ready || m.Error != "" || row != 4 || x < 0 || x >= 28 {
		return false
	}
	if x < 14 {
		m.SelectComment(-1)
	} else {
		m.SelectComment(1)
	}
	return true
}
