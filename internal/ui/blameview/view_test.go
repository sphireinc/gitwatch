package blameview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/blame"
)

func TestSelectionPagingAndSanitizedView(t *testing.T) {
	m := Model{}
	m.SetPage("file.txt", 4, []blame.Line{{FinalSHA: "abcdef123456", Author: "A\nB", Content: []byte("line\x1b[31m")}}, true)
	m.AppendPage(5, []blame.Line{{FinalSHA: "1234567", Content: []byte("next")}}, false)
	m.Move(1)
	if m.Selected != 1 || m.HasMore {
		t.Fatalf("model state = selected %d more %v", m.Selected, m.HasMore)
	}
	view := m.View()
	if !strings.Contains(view, "next") || strings.Contains(view, "\nB") || strings.Contains(view, "\x1b") {
		t.Fatalf("unsafe or missing view content: %q", view)
	}
}
