package githubview

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/provider"
)

func TestSelectedReviewThreadShowsParentAndScrollsWithinViewport(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "o", Name: "r"}, "main", provider.PullRequest{Number: 4}, provider.ChecksSnapshot{})
	m.SetWorkflows([]provider.WorkflowRun{{ID: 304, Name: "CI"}})
	m.SetComments([]provider.ReviewComment{
		{ID: 1, Author: "parent", Body: "original comment"},
		{ID: 2, Author: "reply", InReplyTo: 1, Body: strings.Repeat("long text 界\n", 30) + "final marker\x1b[31m"},
	})
	m.SelectComment(1)
	view := m.ViewWithSize(80, 19)
	for _, want := range []string{"Review thread #2", "Parent #1", "original comment", "Selected #2", "[Prev comment] [Next comment]"} {
		if !strings.Contains(view, want) {
			t.Fatalf("review viewport missing %q:\n%s", want, view)
		}
	}
	m.ScrollReview(10000, 80, 19)
	view = m.ViewWithSize(80, 19)
	if !strings.Contains(view, "final marker�[31m") || strings.Contains(view, "\x1b") || len(strings.Split(view, "\n")) > 19 {
		t.Fatalf("review scroll/sanitization failed:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if runewidth.StringWidth(line) > 80 {
			t.Fatal("review line exceeds viewport")
		}
	}
	if !m.SelectReviewControl(4, 1) || m.SelectedComment != 0 || m.ReviewOffset != 0 {
		t.Fatal("previous-comment mouse control must select and reset scroll")
	}
	m.SelectRun(1)
	if m.ReviewFocused || m.SelectReviewControl(4, 1) {
		t.Fatal("workflow navigation should leave review focus")
	}
}

func TestReviewThreadDoesNotInventMissingParent(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "o", Name: "r"}, "main", provider.PullRequest{Number: 4}, provider.ChecksSnapshot{})
	m.SetComments([]provider.ReviewComment{{ID: 2, InReplyTo: 1, Body: "reply"}})
	m.SelectComment(0)
	if view := m.ViewWithSize(80, 19); !strings.Contains(view, "Reply to #1 (parent not loaded)") {
		t.Fatalf("missing parent should be explicit:\n%s", view)
	}
}
