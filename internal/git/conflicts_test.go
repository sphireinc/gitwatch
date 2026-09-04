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

func TestExternalMergeToolCommandUsesTypedPath(t *testing.T) {
	command, err := NewRunner("/repo").ExternalMergeToolCommand([]byte("space name"))
	if err != nil {
		t.Fatal(err)
	}
	if got := command.Args; len(got) != 5 || got[1] != "mergetool" || got[4] != "space name" {
		t.Fatalf("command args = %#v", got)
	}
}
