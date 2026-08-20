package operations

import (
	"context"
	"testing"
	"time"
)

func TestCommandProducesStructuredMessage(t *testing.T) {
	e := New(1)
	msg := e.Command(context.Background(), "job", "repo", "test", time.Second, func(context.Context) error { return nil })()
	if msg.Result.State != Succeeded || msg.Result.ID != "job" {
		t.Fatal(msg)
	}
}
