// Package layout computes responsive dashboard rectangles.
package layout

// Mode identifies the responsive dashboard arrangement.
type Mode uint8

const (
	// Wide places files and details side by side.
	Wide Mode = iota
	Medium
	Narrow
	TooSmall
)

// Rect is a terminal rectangle in cell coordinates.
type Rect struct{ X, Y, Width, Height int }

// Layout contains the rectangles used by the status dashboard.
type Layout struct {
	Mode                                                          Mode
	Header, Metrics, Files, Details, CommitTree, Activity, Footer Rect
	Message                                                       string
}

// Split contains the configured wide-layout panel percentages.
type Split struct {
	FilesPercent   int
	DetailsPercent int
}

// DefaultSplit returns the standard 60/40 files/details split.
func DefaultSplit() Split {
	return Split{FilesPercent: 60, DetailsPercent: 40}
}

// Compute calculates a layout using the default panel split.
func Compute(width, height int) Layout {
	return ComputeWithSplit(width, height, DefaultSplit())
}

// ComputeWithSplit calculates a layout using split for wide terminals.
func ComputeWithSplit(width, height int, split Split) Layout {
	return ComputeWithSplitAndCommitTree(width, height, split, false)
}

// ComputeWithSplitAndCommitTree calculates a layout with an optional lower history pane.
func ComputeWithSplitAndCommitTree(width, height int, split Split, withCommitTree bool) Layout {
	l := Layout{Mode: Wide}
	if width < 60 || height < 12 {
		l.Mode = TooSmall
		l.Message = "Terminal too small — resize to at least 60×12"
		l.Header = Rect{Width: width, Height: height}
		return l
	}
	l.Header = Rect{X: 0, Y: 0, Width: width, Height: 1}
	l.Metrics = Rect{X: 0, Y: 1, Width: width, Height: 1}
	l.Footer = Rect{X: 0, Y: height - 1, Width: width, Height: 1}
	l.Activity = Rect{X: 0, Y: height - 4, Width: width, Height: 3}
	contentTop, contentHeight := 3, height-7
	if width >= 140 {
		filesWidth, detailsWidth := splitWidths(width, split)
		filesHeight, treeHeight := splitTreeHeights(contentHeight, withCommitTree)
		l.Files = Rect{X: 0, Y: contentTop, Width: filesWidth, Height: filesHeight}
		l.CommitTree = Rect{X: 0, Y: contentTop + filesHeight, Width: filesWidth, Height: treeHeight}
		l.Details = Rect{X: filesWidth, Y: contentTop, Width: detailsWidth, Height: contentHeight}
	} else if width >= 90 {
		l.Mode = Medium
		filesHeight, treeHeight := splitTreeHeights(contentHeight, withCommitTree)
		l.Files = Rect{X: 0, Y: contentTop, Width: width, Height: filesHeight}
		l.CommitTree = Rect{X: 0, Y: contentTop + filesHeight, Width: width, Height: treeHeight}
		l.Details = Rect{X: 0, Y: contentTop, Width: width, Height: contentHeight}
	} else {
		l.Mode = Narrow
		filesHeight, treeHeight := splitTreeHeights(contentHeight, withCommitTree)
		l.Files = Rect{X: 0, Y: contentTop, Width: width, Height: filesHeight}
		l.CommitTree = Rect{X: 0, Y: contentTop + filesHeight, Width: width, Height: treeHeight}
		l.Details = l.Files
	}
	return l
}

func splitTreeHeights(contentHeight int, enabled bool) (int, int) {
	if !enabled || contentHeight < 6 {
		return contentHeight, 0
	}
	tree := max(3, contentHeight/4)
	return contentHeight - tree, tree
}

func splitWidths(width int, split Split) (int, int) {
	if split.FilesPercent <= 0 || split.DetailsPercent <= 0 || split.FilesPercent+split.DetailsPercent != 100 {
		split = DefaultSplit()
	}
	filesWidth := width * split.FilesPercent / 100
	filesWidth = max(1, min(width-1, filesWidth))
	return filesWidth, width - filesWidth
}

// Contains reports whether a terminal coordinate is inside the rectangle.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}
