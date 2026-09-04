package manage

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
)

func TestPlanAndApplyMultipleTemplatesPreservesPrefix(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	before := []byte("# hand written\n!keep.txt")
	snapshot, err := domain.NewDocumentSnapshot("repo", root, ".gitignore", before, 0644)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanAddTemplates(snapshot, cat, []domain.TemplateID{"root/Go", "root/Node"})
	if err != nil {
		t.Fatal(err)
	}
	if string(plan.ResultBytes[:len(before)]) != string(before) {
		t.Fatal("existing prefix changed")
	}
	if len(plan.Selected) != 2 || plan.Selected[0] != "root/Go" {
		t.Fatalf("selection order=%v", plan.Selected)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), before, 0644); err != nil {
		t.Fatal(err)
	}
	if err := Apply(plan); err != nil {
		t.Fatal(err)
	}
	result, _ := os.ReadFile(filepath.Join(root, ".gitignore"))
	if string(result[:len(before)]) != string(before) {
		t.Fatal("applied prefix changed")
	}
}

func TestApplyRejectsConcurrentEdit(t *testing.T) {
	cat, _ := catalog.Default()
	root := t.TempDir()
	path := filepath.Join(root, ".gitignore")
	before := []byte("# original")
	os.WriteFile(path, before, 0644)
	snapshot, _ := domain.NewDocumentSnapshot("repo", root, ".gitignore", before, 0644)
	plan, err := PlanAddTemplates(snapshot, cat, []domain.TemplateID{"root/Go"})
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte("# external"), 0644)
	if err := Apply(plan); !errors.Is(err, domain.ErrConcurrentModification) {
		t.Fatalf("error=%v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "# external" {
		t.Fatal("concurrent edit overwritten")
	}
}

func TestPlanUsesCRLFAndDoesNotDuplicateManagedTemplate(t *testing.T) {
	cat, _ := catalog.Default()
	root := t.TempDir()
	before := []byte("# comment\r\n!keep.txt\r\n")
	snapshot, _ := domain.NewDocumentSnapshot("repo", root, ".gitignore", before, 0644)
	plan, err := PlanAddTemplates(snapshot, cat, []domain.TemplateID{"root/Go"})
	if err != nil {
		t.Fatal(err)
	}
	if string(plan.ResultBytes[:len(before)]) != string(before) || !containsBytes(plan.ResultBytes, []byte("\r\n# >>> gitwatch:gitignore begin")) {
		t.Fatal("CRLF boundary was not preserved")
	}
	goTemplate, _ := cat.Get("root/Go")
	managedBytes, _ := managed.EncodeManagedBlock(goTemplate.ID, "github/gitignore", cat.Version(), goTemplate.ContentSHA256, goTemplate.Content, []byte("\n"))
	managedSnapshot, _ := domain.NewDocumentSnapshot("repo", root, ".gitignore", managedBytes, 0644)
	if _, err := PlanAddTemplates(managedSnapshot, cat, []domain.TemplateID{goTemplate.ID}); !errors.Is(err, ErrAlreadyInstalled) {
		t.Fatalf("duplicate error=%v", err)
	}
}

func containsBytes(value, needle []byte) bool { return bytes.Contains(value, needle) }
