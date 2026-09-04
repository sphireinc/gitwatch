package git

import (
	"context"
	"testing"
)

func TestResolveConflictValidatesActionAndPath(t *testing.T) {
	runner := NewRunner(t.TempDir())
	if _, err := runner.ResolveConflict(context.Background(), nil, ChooseOurs); err == nil {
		t.Fatal("expected missing path error")
	}
	if _, err := runner.ResolveConflict(context.Background(), []byte("file"), "invalid"); err == nil {
		t.Fatal("expected unsupported action error")
	}
}
