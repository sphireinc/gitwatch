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

func Compute(width, height int) Layout {
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
	contentTop, contentHeight := 2, height-6
	if width >= 140 {
		split := width * 3 / 5
		l.Files = Rect{X: 0, Y: contentTop, Width: split, Height: contentHeight}
		l.Details = Rect{X: split, Y: contentTop, Width: width - split, Height: contentHeight}
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

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}
