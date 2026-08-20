package githubview

import (
	"strings"
	"testing"

	"github.com/jusanchez/gitwatch/internal/provider"
)

func TestViewRendersPullRequestAndChecksSafely(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Title: "Improve", State: "open", Head: "feature", Base: "main", Mergeable: "clean", URL: "https://github.com/octo/repo/pull/4"}, provider.ChecksSnapshot{Passing: 2, Failing: 1, Pending: 1})
	view := m.View()
	for _, want := range []string{"octo/repo", "PR #4", "Checks: 2 passing  1 failing  1 pending"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q: %s", want, view)
		}
	}
	unsafe := New()
	unsafe.SetError(provider.Repository{Owner: "octo\x1b", Name: "repo"}, "main", nil)
	if strings.Contains(unsafe.View(), "\x1b") {
		t.Fatal("provider error view contains escape")
	}
}
