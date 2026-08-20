package mouse

import "github.com/jusanchez/gitwatch/internal/ui/layout"

type Action uint8

const (
	None Action = iota
	SelectRow
	ToggleStage
	OpenDiff
	ScrollUp
	ScrollDown
)

type HitMap struct {
	Files                     layout.Rect
	RowTop, RowHeight, Offset int
	StageX, StageWidth        int
	RowCount                  int
}

func (h HitMap) Hit(x, y int, wheel int) (Action, int, bool) {
	if wheel < 0 && h.Files.Contains(x, y) {
		return ScrollDown, -1, true
	}
	if wheel > 0 && h.Files.Contains(x, y) {
		return ScrollUp, -1, true
	}
	if !h.Files.Contains(x, y) {
		return None, -1, false
	}
	row := y - h.RowTop + h.Offset
	if row < 0 || row >= h.RowCount || h.RowHeight <= 0 {
		return None, -1, false
	}
	if x >= h.StageX && x < h.StageX+h.StageWidth {
		return ToggleStage, row, true
	}
	return SelectRow, row, true
}

func DoubleClick() Action { return OpenDiff }
