package layout

type Mode uint8

const (
	Wide Mode = iota
	Medium
	Narrow
	TooSmall
)

type Rect struct{ X, Y, Width, Height int }
type Layout struct {
	Mode                                              Mode
	Header, Metrics, Files, Details, Activity, Footer Rect
	Message                                           string
}

type Split struct {
	FilesPercent   int
	DetailsPercent int
}

func DefaultSplit() Split {
	return Split{FilesPercent: 60, DetailsPercent: 40}
}

func Compute(width, height int) Layout {
	return ComputeWithSplit(width, height, DefaultSplit())
}

func ComputeWithSplit(width, height int, split Split) Layout {
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
		l.Files = Rect{X: 0, Y: contentTop, Width: filesWidth, Height: contentHeight}
		l.Details = Rect{X: filesWidth, Y: contentTop, Width: detailsWidth, Height: contentHeight}
	} else if width >= 90 {
		l.Mode = Medium
		l.Files = Rect{X: 0, Y: contentTop, Width: width, Height: contentHeight}
		l.Details = Rect{X: 0, Y: contentTop, Width: width, Height: contentHeight}
	} else {
		l.Mode = Narrow
		l.Files = Rect{X: 0, Y: contentTop, Width: width, Height: contentHeight}
		l.Details = l.Files
	}
	return l
}

func splitWidths(width int, split Split) (int, int) {
	if split.FilesPercent <= 0 || split.DetailsPercent <= 0 || split.FilesPercent+split.DetailsPercent != 100 {
		split = DefaultSplit()
	}
	filesWidth := width * split.FilesPercent / 100
	filesWidth = max(1, min(width-1, filesWidth))
	return filesWidth, width - filesWidth
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}
