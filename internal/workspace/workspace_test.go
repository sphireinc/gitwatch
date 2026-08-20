package workspace

import (
	"context"
	"testing"
)

func TestNavigationAndBreadcrumbsAreIndependentSnapshots(t *testing.T) {
	model := New()
	model.Navigate(Log, "History")
	current, breadcrumbs, _, _ := model.Snapshot()
	if current != Log || len(breadcrumbs) != 2 || breadcrumbs[1].Label != "History" {
		t.Fatalf("unexpected navigation snapshot: %v %#v", current, breadcrumbs)
	}
	breadcrumbs[0].Label = "mutated"
	_, fresh, _, _ := model.Snapshot()
	if fresh[0].Label != "Status" {
		t.Fatal("breadcrumb snapshot aliases model state")
	}
	model.Back()
	current, _, _, _ = model.Snapshot()
	if current != Status {
		t.Fatalf("back did not restore status: %s", current)
	}
}

func TestJobCancellationAndCompletionState(t *testing.T) {
	model := New()
	jobContext := model.StartJob(context.Background(), "refresh", "Refresh")
	_, _, _, jobs := model.Snapshot()
	if jobs["refresh"].State != JobRunning {
		t.Fatalf("job did not start: %#v", jobs["refresh"])
	}
	jobs["refresh"].Cancel()
	<-jobContext.Done()
	model.FinishJob("refresh", context.Canceled)
	_, _, _, jobs = model.Snapshot()
	if jobs["refresh"].State != JobCancelled {
		t.Fatalf("job was not cancelled: %#v", jobs["refresh"])
	}
}
