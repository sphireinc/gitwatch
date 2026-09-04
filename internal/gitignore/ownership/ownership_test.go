package ownership

import (
	"testing"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/document"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/managed"
)

func TestManagedRemovalOnlyRemovesSelectedBlock(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	node, _ := cat.Get("root/Node")
	goTemplate, _ := cat.Get("root/Go")
	first, _ := managed.EncodeManagedBlock(node.ID, "x", "x", node.ContentSHA256, node.Content, []byte("\n"))
	second, _ := managed.EncodeManagedBlock(goTemplate.ID, "x", "x", goTemplate.ContentSHA256, goTemplate.Content, []byte("\n"))
	doc, err := document.Parse(append(first, second...))
	if err != nil {
		t.Fatal(err)
	}
	index := Build(doc, []catalog.Template{node, goTemplate}, []domain.TemplateID{node.ID, goTemplate.ID})
	decisions := index.Removal([]domain.TemplateID{node.ID}, []domain.TemplateID{node.ID, goTemplate.ID})
	for _, decision := range decisions {
		if decision.Rule.Kind == ManagedOccurrence && decision.Rule.ManagedBy == node.ID && !decision.SafeToRemove {
			t.Fatalf("selected block not removable: %+v", decision)
		}
		if decision.Rule.Kind == ManagedOccurrence && decision.Rule.ManagedBy == goTemplate.ID && decision.SafeToRemove {
			t.Fatalf("unselected block marked removable: %+v", decision)
		}
	}
}

func TestDuplicateUnmanagedRuleIsAmbiguous(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	node, _ := cat.Get("root/Node")
	doc, _ := document.Parse([]byte("node_modules/\nnode_modules/\n"))
	index := Build(doc, []catalog.Template{node}, nil)
	decisions := index.Removal([]domain.TemplateID{node.ID}, nil)
	for _, decision := range decisions {
		if decision.Rule.Text == "node_modules/" && decision.SafeToRemove {
			t.Fatal("duplicate handwritten rule marked safe")
		}
	}
}

func TestSharedUnmanagedRuleRequiresRetainingUnselectedOwner(t *testing.T) {
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	node, _ := cat.Get("root/Node")
	goTemplate, _ := cat.Get("root/Go")
	shared := ""
	for rule := range significant(node.Content) {
		if _, ok := significant(goTemplate.Content)[rule]; ok {
			shared = rule
			break
		}
	}
	if shared == "" {
		t.Fatal("fixtures have no shared rule")
	}
	doc, _ := document.Parse([]byte(shared + "\n"))
	index := Build(doc, []catalog.Template{node, goTemplate}, []domain.TemplateID{node.ID, goTemplate.ID})
	decisions := index.Removal([]domain.TemplateID{node.ID}, []domain.TemplateID{node.ID, goTemplate.ID})
	if decisions[0].SafeToRemove {
		t.Fatalf("shared rule was removable: %+v", decisions[0])
	}
	decisions = index.Removal([]domain.TemplateID{node.ID, goTemplate.ID}, []domain.TemplateID{node.ID, goTemplate.ID})
	if !decisions[0].SafeToRemove {
		t.Fatalf("joint removal retained unique shared rule: %+v", decisions[0])
	}
	if len(index.Overlaps()) == 0 {
		t.Fatal("overlap was not exposed")
	}
}
