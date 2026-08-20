package app

import "testing"

func TestMotionModes(t *testing.T) {
	if !MotionFull.Ticks() || MotionReduced.HighlightDuration() >= MotionFull.HighlightDuration() || MotionOff.Ticks() {
		t.Fatal("motion policy")
	}
}
