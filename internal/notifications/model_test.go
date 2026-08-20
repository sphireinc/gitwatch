package notifications

import (
	"testing"
	"time"
)

func TestModelBoundsAndDismissesAttention(t *testing.T) {
	model := New(3, false)
	first := model.Add(Notification{Kind: Conflict, Level: Warning, Title: "Conflict", Attention: true, At: time.Unix(1, 0)})
	model.Add(Notification{Kind: JobComplete, Level: Success, Title: "Done", At: time.Unix(2, 0)})
	model.Add(Notification{Kind: RemoteStale, Level: Warning, Title: "Stale", Attention: true, At: time.Unix(3, 0)})
	if len(model.Items()) != 3 || model.Attention() != 2 {
		t.Fatalf("unexpected bounded model: %#v", model.Items())
	}
	if !model.Dismiss(first) || model.Dismiss(first) {
		t.Fatal("dismissal did not behave idempotently")
	}
}

func TestQuietModeSuppressesAttention(t *testing.T) {
	model := New(10, true)
	model.Add(Notification{Kind: PushFailure, Level: Error, Attention: true})
	if model.Attention() != 0 {
		t.Fatal("quiet mode retained attention")
	}
}
