package git

import (
	"context"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/conflicts"
	"github.com/sphireinc/git-watch/internal/sequencer"
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

func TestResolveConflictSupportsBothChoice(t *testing.T) {
	if ChooseBoth != ConflictChoice("both") {
		t.Fatal("both choice identity changed")
	}
}

func TestOperationLifecycleRejectsUnsupportedSkip(t *testing.T) {
	if _, err := NewRunner(t.TempDir()).OperationLifecycle(context.Background(), sequencer.KindMerge, "skip"); err == nil {
		t.Fatal("merge skip should be rejected")
	}
	if _, err := NewRunner(t.TempDir()).OperationLifecycle(context.Background(), sequencer.KindUnknown, "continue"); err == nil {
		t.Fatal("unknown operation should be rejected")
	}
}

func TestResolveConflictRegionRejectsStaleAndDoesNotStage(t *testing.T) {
	dir := t.TempDir()
	runner := NewRunner(dir)
	if _, err := runner.Run(context.Background(), "init", "--quiet"); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "file")
	data := []byte("before\n<<<<<<< ours\none\n=======\ntwo\n>>>>>>> theirs\nafter\n")
	if err := os.WriteFile(path, data, 0o751); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if err := os.WriteFile(path, []byte("external\n"), 0o751); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ResolveConflictRegion(context.Background(), []byte("file"), hash, 0, conflicts.ChoiceOurs, nil); err != conflicts.ErrStaleDocument {
		t.Fatalf("stale error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o751); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.ResolveConflictRegion(context.Background(), []byte("file"), hash, 0, conflicts.ChoiceTheirs, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != "before\ntwo\nafter\n" {
		t.Fatalf("updated file = %q", updated)
	}
	status, err := runner.Run(context.Background(), "status", "--porcelain=v1", "-z")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(status.Stdout), "A  ") {
		t.Fatalf("region edit implicitly staged: %q", status.Stdout)
	}
}

func TestResolveMultipleConflictFilesAndRefreshAuthoritativeState(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	runner := NewRunner(dir)
	for _, args := range [][]string{{"init", "-b", "main", "--", dir}, {"config", "user.name", "test"}, {"config", "user.email", "test@example.com"}} {
		if _, err := runner.Run(ctx, args...); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"one", "two"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("base\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte(name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runner.Commit(ctx, CommitOptions{Message: []byte("base\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "-c", "feature"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one", "two"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("theirs\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte(name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runner.Commit(ctx, CommitOptions{Message: []byte("feature\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "switch", "main"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one", "two"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("ours\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := runner.Stage(ctx, []byte(name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := runner.Commit(ctx, CommitOptions{Message: []byte("main\n")}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(ctx, "merge", "feature"); err == nil {
		t.Fatal("expected merge conflict")
	}
	discovery, err := Discover(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := Snapshot(ctx, discovery, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Conflicts) != 2 {
		t.Fatalf("conflicts = %#v", snapshot.Conflicts)
	}
	for _, conflict := range snapshot.Conflicts {
		if _, err := runner.ResolveConflict(ctx, conflict.Path, ChooseOurs); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"one", "two"} {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "ours\n" {
			t.Fatalf("%s content = %q", name, content)
		}
	}
	for _, conflict := range snapshot.Conflicts {
		if _, err := runner.ResolveConflict(ctx, conflict.Path, MarkResolved); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err = Snapshot(ctx, discovery, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Conflicts) != 0 || snapshot.Counts.Conflicted != 0 {
		t.Fatalf("resolved snapshot = %+v", snapshot)
	}
	status, err := runner.Run(ctx, "status", "--porcelain=v1", "-z")
	if err != nil || strings.Contains(string(status.Stdout), "UU") {
		t.Fatalf("status after resolution = %q err=%v", status.Stdout, err)
	}
}
