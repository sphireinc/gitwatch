package commitview

import (
	"github.com/jusanchez/gitwatch/internal/commitmodel"
	"strings"
	"testing"
)

func TestComposerReadinessAndView(t *testing.T) {
	c := New([]commitmodel.File{{Path: "a", Staged: true}})
	if c.Ready() {
		t.Fatal("empty composer ready")
	}
	c.SetSubject("add feature")
	if !c.Ready() || !strings.Contains(c.View(), "Ready to commit") {
		t.Fatal(c.View())
	}
}
