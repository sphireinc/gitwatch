package githubview

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/sphireinc/git-watch/internal/provider"
)

func TestWorkflowViewShowsMetadataAndKeepsSelectionWithinViewport(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "o", Name: "r"}, "main", provider.PullRequest{}, provider.ChecksSnapshot{})
	runs := make([]provider.WorkflowRun, 100)
	for i := range runs {
		runs[i] = provider.WorkflowRun{ID: int64(300 + i), Name: fmt.Sprintf("workflow-%03d", i), Status: "completed", Conclusion: "failure", Attempt: 3, URL: fmt.Sprintf("https://github.com/o/r/actions/runs/%d", 300+i), StartedAt: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 9, 28, 0, 2, 0, 0, time.UTC)}
	}
	m.SetWorkflows(runs)
	m.SelectedRun = 70
	view := m.ViewWithSize(80, 19)
	if !strings.Contains(view, "> workflow-070 [completed/failure] attempt:3 elapsed:2m0s") {
		t.Fatalf("selected workflow metadata not visible:\n%s", view)
	}
	if lines := strings.Split(view, "\n"); len(lines) > 19 {
		t.Fatalf("rendered %d lines", len(lines))
	}
	for _, line := range strings.Split(view, "\n") {
		if runewidth.StringWidth(line) > 80 {
			t.Fatalf("line exceeds viewport: %q", line)
		}
	}
	start, _ := m.workflowWindow(19)
	if !m.SelectWorkflowRow(5, 19) || m.SelectedRun != start {
		t.Fatalf("mouse row should select first visible workflow: %d, want %d", m.SelectedRun, start)
	}
	if m.SelectWorkflowRow(4, 19) || m.SelectWorkflowRow(100, 19) {
		t.Fatal("non-workflow rows must not select")
	}
}

func TestWorkflowSelectionSurvivesReorderAndFallsBackWhenRemoved(t *testing.T) {
	m := New()
	m.SetWorkflows([]provider.WorkflowRun{{ID: 30}, {ID: 40}})
	m.SelectedRun = 1
	m.SetWorkflows([]provider.WorkflowRun{{ID: 40}, {ID: 30}})
	if m.SelectedRun != 0 || m.Workflows[m.SelectedRun].ID != 40 {
		t.Fatal("refresh lost selected workflow identity")
	}
	m.SetWorkflows([]provider.WorkflowRun{{ID: 30}})
	if m.SelectedRun != 0 {
		t.Fatal("removed workflow selection must fall back safely")
	}
}

func TestWorkflowTextCannotInjectViewportRowsOrControls(t *testing.T) {
	m := New()
	m.SetData(provider.Repository{Owner: "o", Name: "r"}, "main", provider.PullRequest{}, provider.ChecksSnapshot{})
	m.SetWorkflows([]provider.WorkflowRun{{ID: 30, Name: "name\nforged\t\x1b[31m", URL: "https://github.com/o/r/actions/runs/30\nforged"}})
	view := m.ViewWithSize(80, 19)
	if strings.Contains(view, "\x1b") || strings.Contains(view, "\t") || strings.Contains(view, "\nforged") {
		t.Fatalf("untrusted workflow text affected rows/terminal: %q", view)
	}
}
