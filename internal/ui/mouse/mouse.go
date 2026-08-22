package mouse

import "github.com/sphireinc/git-watch/internal/ui/layout"

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
	RowHeights                []int
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
	displayRow := y - h.RowTop
	if displayRow < 0 || h.RowHeight <= 0 {
		return None, -1, false
	}
	row := displayRow + h.Offset
	if len(h.RowHeights) > 0 {
		row = h.Offset
		for _, rowHeight := range h.RowHeights {
			if displayRow < rowHeight {
				break
			}
			displayRow -= rowHeight
			row++
		}
		if displayRow < 0 || row >= h.RowCount || rowHeightAt(h.RowHeights, row-h.Offset) <= 0 {
			return None, -1, false
		}
	} else if row < 0 || row >= h.RowCount {
		return None, -1, false
	}
	if x >= h.StageX && x < h.StageX+h.StageWidth {
		return ToggleStage, row, true
	}
	return SelectRow, row, true
}

func rowHeightAt(heights []int, index int) int {
	if index < 0 || index >= len(heights) {
		return 0
	}
	return heights[index]
}

func DoubleClick() Action { return OpenDiff }
