package multirepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

func TestBatchPlanAndApplyIsExplicitAndFailureIsolated(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	success, concurrent, readonly, symlink := filepath.Join(root, "success"), filepath.Join(root, "concurrent"), filepath.Join(root, "readonly"), filepath.Join(root, "symlink")
	for _, path := range []string{success, concurrent, readonly, symlink} {
		if err := os.Mkdir(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(concurrent, ".gitignore"), []byte("# before\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(readonly, ".gitignore"), []byte("# read-only\n"), 0444); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(success, ".gitignore"), filepath.Join(symlink, ".gitignore")); err != nil {
		t.Fatal(err)
	}
	repositories := []Repository{{ID: "success", Root: success}, {ID: "concurrent", Root: concurrent}, {ID: "readonly", Root: readonly}, {ID: "symlink", Root: symlink}}
	plans := PlanAdd(context.Background(), repositories, cat, []domain.TemplateID{"root/Go"})
	if len(plans) != 4 || plans[0].Preview.Diff == "" || !plans[2].Skipped || !plans[3].Skipped {
		t.Fatalf("plans = %#v", plans)
	}
	if err := os.WriteFile(filepath.Join(concurrent, ".gitignore"), []byte("# changed\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var refreshed atomic.Int32
	results := Apply(context.Background(), plans, 2, func(context.Context, Repository) error { refreshed.Add(1); return nil })
	if results[0].Status != "succeeded" || results[1].Status != "failed" || results[2].Status != "skipped" || results[3].Status != "skipped" {
		t.Fatalf("results = %#v", results)
	}
	if !errors.Is(results[1].Err, domain.ErrConcurrentModification) || refreshed.Load() != 1 {
		t.Fatalf("isolation = results=%#v refreshed=%d", results, refreshed.Load())
	}
	if _, err := os.Stat(filepath.Join(success, ".gitignore")); err != nil {
		t.Fatal(err)
	}
}
