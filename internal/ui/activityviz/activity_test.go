package activityviz

import (
	"testing"
	"time"
)

func TestHeatLevelWeightsConflicts(t *testing.T) {
	if HeatLevel(0, 0, 0, 0) != 0 || HeatLevel(0, 0, 1, 0) != 1 || HeatLevel(0, 0, 0, 1) < 2 {
		t.Fatalf("unexpected heat levels")
	}
}

func TestBarIsFixedWidthAndClamped(t *testing.T) {
	if got := Bar(2, 4, 6); got != "###..." {
		t.Fatalf("Bar = %q", got)
	}
	if got := Bar(99, 1, 4); got != "####" {
		t.Fatalf("clamped Bar = %q", got)
	}
}

func TestSparklineKeepsRecentBoundedValues(t *testing.T) {
	if got := Sparkline([]int{1, 2, 3, 4}, 3); got != "▄▆█" {
		t.Fatalf("Sparkline = %q", got)
	}
	if got := Sparkline(nil, 4); got != "····" {
		t.Fatalf("empty Sparkline = %q", got)
	}
}

func TestCommitBucketsAreBoundedAndIgnoreOutOfWindowValues(t *testing.T) {
	now := time.Unix(10*24*3600, 0)
	got := CommitBuckets([]int64{now.Unix(), now.Add(-24 * time.Hour).Unix(), now.Add(-10 * 24 * time.Hour).Unix(), now.Add(time.Hour).Unix()}, now, 24*time.Hour, 3)
	if len(got) != 3 || got[2] != 1 || got[1] != 1 || got[0] != 0 {
		t.Fatalf("commit buckets = %#v", got)
	}
}
