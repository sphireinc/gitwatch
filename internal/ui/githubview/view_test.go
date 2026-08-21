package githubview

import (
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/provider"
)

func TestViewRendersPullRequestAndChecksSafely(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "octo", Name: "repo"}, "main", provider.PullRequest{Number: 4, Title: "Improve", State: "open", Head: "feature", Base: "main", Mergeable: "clean", URL: "https://github.com/octo/repo/pull/4"}, provider.ChecksSnapshot{Passing: 2, Failing: 1, Pending: 1, Runs: []provider.CheckRun{{Name: "build", Status: "completed", Conclusion: "failure", URL: "https://github.com/check/1", Failure: "compile failed"}}})
	view := m.View()
	for _, want := range []string{"octo/repo", "PR #4", "Checks: 2 passing  1 failing  1 pending", "check build", "compile failed", "https://github.com/check/1"} {
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
