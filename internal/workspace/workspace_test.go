package workspace

import (
	"context"
	"testing"
)

func TestNavigationAndCancellableJobs(t *testing.T) {
	m := New()
	m.Navigate(Log, "History")
	m.Navigate(Remotes, "Remotes")
	m.Back()
	v, b, _, _ := m.Snapshot()
	if v != Log || len(b) != 2 {
		t.Fatal(v, b)
	}
	ctx := m.StartJob(context.Background(), "fetch", "fetch")
	m.FinishJob("fetch", nil)
	if ctx.Err() != nil {
		t.Fatal(ctx.Err())
	}
	_, _, _, jobs := m.Snapshot()
	if jobs["fetch"].State != JobSucceeded {
		t.Fatal(jobs)
	}
}
