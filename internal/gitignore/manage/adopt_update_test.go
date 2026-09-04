package manage

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
)

func TestPlanAdoptWrapsOnlyOneExactContiguousSegment(t *testing.T) {
	cat, _ := catalog.Default()
	template, _ := cat.Get("root/Go")
	before := append([]byte("handwritten\n"), template.Content...)
	snapshot, _ := domain.NewDocumentSnapshot("repo", t.TempDir(), ".gitignore", before, 0644)
	plan, err := PlanAdoptTemplate(snapshot, cat, template.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(plan.ResultBytes, template.Content) || !bytes.Contains(plan.ResultBytes, []byte("id=root/Go")) {
		t.Fatal("adoption did not preserve exact segment")
	}
	if !bytes.Contains(plan.ResultBytes, []byte("handwritten\n")) {
		t.Fatal("handwritten content changed")
	}
}

func TestPlanAdoptRefusesScatteredMatch(t *testing.T) {
	cat, _ := catalog.Default()
	template, _ := cat.Get("root/Go")
	lines := bytes.Split(template.Content, []byte("\n"))
	if len(lines) < 4 {
		t.Fatal("fixture template too short")
	}
	before := append([]byte(nil), lines[0]...)
	before = append(before, []byte("\n# between\n")...)
	before = append(before, bytes.Join(lines[1:], []byte("\n"))...)
	snapshot, _ := domain.NewDocumentSnapshot("repo", t.TempDir(), ".gitignore", before, 0644)
	if _, err := PlanAdoptTemplate(snapshot, cat, template.ID); !errors.Is(err, ErrNoAdoptableSegment) {
		t.Fatalf("adopt error=%v", err)
	}
}

func TestPlanUpdateReplacesOneStaleBlockAndPreservesHandwrittenContent(t *testing.T) {
	cat, _ := catalog.Default()
	template, _ := cat.Get("root/Go")
	stale, err := managed.EncodeManagedBlock(template.ID, "github/gitignore", "oldcommit", "oldhash", []byte("old-rule\n"), []byte("\n"))
	if err != nil {
		t.Fatal(err)
	}
	before := append([]byte("before\n"), stale...)
	before = append(before, []byte("after\n")...)
	snapshot, _ := domain.NewDocumentSnapshot("repo", t.TempDir(), ".gitignore", before, 0644)
	plan, err := PlanUpdateTemplates(snapshot, cat, []domain.TemplateID{template.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(plan.ResultBytes, template.Content) || !bytes.Contains(plan.ResultBytes, []byte("before\n")) || !bytes.Contains(plan.ResultBytes, []byte("after\n")) || bytes.Contains(plan.ResultBytes, []byte("old-rule")) {
		t.Fatalf("stale block update was not block-local: %q", plan.ResultBytes)
	}
}

func TestSummarizeMixedActions(t *testing.T) {
	summary := SummarizeMixed([]MixedAction{{Kind: ActionAdd}, {Kind: ActionAdd}, {Kind: ActionRemove}, {Kind: ActionUpdate}})
	if summary.String() != "2 add, 1 remove, 1 update" {
		t.Fatalf("summary=%q", summary.String())
	}
}
