package manage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

func TestPlanCreateTemplatesSupportsMultipleTemplatesWithoutExistingFile(t *testing.T) {
	root := t.TempDir()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := domain.NewDocumentSnapshot("repo-a", root, ".gitignore", nil, 0644)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanCreateTemplates(snapshot, cat, []domain.TemplateID{"root/Go", "global/macOS"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Kind != domain.MutationCreate || len(plan.Selected) != 2 || len(plan.ResultBytes) == 0 {
		t.Fatalf("create plan = %+v", plan)
	}
	if err := Create(plan); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(plan.ResultBytes) {
		t.Fatal("created bytes differ from preview")
	}
}

func TestCreateRejectsConcurrentExternalCreation(t *testing.T) {
	root := t.TempDir()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := domain.NewDocumentSnapshot("repo-a", root, ".gitignore", nil, 0644)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanCreateTemplates(snapshot, cat, []domain.TemplateID{"root/Go"})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".gitignore")
	if err := os.WriteFile(path, []byte("# external\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Create(plan); !errors.Is(err, domain.ErrConcurrentModification) {
		t.Fatalf("create error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "# external\n" {
		t.Fatal("external file was overwritten")
	}
}
