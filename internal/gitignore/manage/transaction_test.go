package manage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

func TestPreviewAndUndoGuardAgainstExternalEdit(t *testing.T) {
	cat, _ := catalog.Default()
	root := t.TempDir()
	path := filepath.Join(root, ".gitignore")
	before := []byte("# hand\n")
	snapshot, _ := domain.NewDocumentSnapshot("repo", root, ".gitignore", before, 0644)
	plan, err := PlanAddTemplates(snapshot, cat, []domain.TemplateID{"root/Go"})
	if err != nil {
		t.Fatal(err)
	}
	preview := PreviewPlan(plan)
	if !strings.Contains(preview.Diff, "gitwatch:gitignore") || len(preview.Selected) != 1 {
		t.Fatalf("preview=%+v", preview)
	}
	os.WriteFile(path, before, 0644)
	record, err := ApplyTransaction(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !record.Success || record.AfterSHA256 == "" {
		t.Fatalf("record=%+v", record)
	}
	if err := Undo(record); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(path)
	if string(restored) != string(before) {
		t.Fatalf("restored=%q", restored)
	}
	ApplyTransaction(plan)
	os.WriteFile(path, []byte("external"), 0644)
	if err := Undo(record); !errors.Is(err, ErrUndoConflict) {
		t.Fatalf("undo error=%v", err)
	}
}

func TestApplyRefusesSymlinkAndCleansFailedTempCreation(t *testing.T) {
	cat, _ := catalog.Default()
	root := t.TempDir()
	target := filepath.Join(root, "real")
	os.WriteFile(target, []byte("safe"), 0644)
	link := filepath.Join(root, ".gitignore")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	snapshot, _ := domain.NewDocumentSnapshot("repo", root, ".gitignore", []byte("safe"), 0644)
	plan, _ := PlanAddTemplates(snapshot, cat, []domain.TemplateID{"root/Go"})
	if err := Apply(plan); !errors.Is(err, domain.ErrUnsafeTarget) {
		t.Fatalf("symlink error=%v", err)
	}
	badRoot := filepath.Join(root, "missing")
	badPlan := plan
	badPlan.Root = badRoot
	if err := Apply(badPlan); err == nil {
		t.Fatal("missing root unexpectedly accepted")
	}
	if matches, _ := filepath.Glob(filepath.Join(root, ".gitignore.gitwatch-*")); len(matches) != 0 {
		t.Fatalf("temporary files leaked: %v", matches)
	}
}
