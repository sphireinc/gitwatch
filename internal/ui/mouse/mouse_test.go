package mouse

import (
	"github.com/sphireinc/git-watch/internal/ui/layout"
	"testing"
)

func TestHitMapSeparatesRowAndStageControl(t *testing.T) {
	h := HitMap{Files: layout.Rect{X: 0, Y: 2, Width: 100, Height: 10}, RowTop: 3, RowHeight: 1, Offset: 4, StageX: 0, StageWidth: 3, RowCount: 20}
	if action, row, ok := h.Hit(10, 5, 0); !ok || action != SelectRow || row != 6 {
		t.Fatal(action, row, ok)
	}
	if action, row, ok := h.Hit(1, 5, 0); !ok || action != ToggleStage || row != 6 {
		t.Fatal(action, row, ok)
	}
	if action, _, ok := h.Hit(10, 5, 1); !ok || action != ScrollUp {
		t.Fatal(action, ok)
	}
	if DoubleClick() != OpenDiff {
		t.Fatal("double click should only open inspection")
	}
}
