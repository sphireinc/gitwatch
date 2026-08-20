package app

type Motion string

const (
	MotionFull    Motion = "full"
	MotionReduced Motion = "reduced"
	MotionOff     Motion = "off"
)

func (m Motion) Ticks() bool { return m != MotionOff }
func (m Motion) HighlightDuration() int {
	if m == MotionFull {
		return 900
	}
	if m == MotionReduced {
		return 250
	}
	return 0
}
