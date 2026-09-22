// Package activityviz provides bounded, terminal-safe activity projections.
package activityviz

import (
	"strings"
	"time"
)

var levels = []rune("▁▂▃▄▅▆▇█")

// HeatLevel maps measurable repository changes to a small stable intensity.
// Conflicts are weighted more heavily because they require immediate attention.
func HeatLevel(staged, unstaged, untracked, conflicts int) int {
	score := max(0, staged) + max(0, unstaged) + max(0, untracked) + max(0, conflicts)*4
	switch {
	case score == 0:
		return 0
	case score <= 2:
		return 1
	case score <= 6:
		return 2
	case score <= 12:
		return 3
	default:
		return 4
	}
}

// HeatGlyph returns a visible intensity glyph. Numeric counts should be shown
// alongside it so color or glyph support is never the only semantic channel.
func HeatGlyph(level int) string {
	if level < 0 {
		level = 0
	}
	if level >= len(levels) {
		level = len(levels) - 1
	}
	return string(levels[level])
}

// Bar returns a fixed-width ASCII bar for a bounded numeric measure.
func Bar(value, maximum, width int) string {
	if width < 1 {
		return ""
	}
	if width > 40 {
		width = 40
	}
	if value < 0 {
		value = 0
	}
	if maximum < 1 {
		maximum = 1
	}
	filled := value * width / maximum
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat(".", width-filled)
}

// Sparkline renders at most width recent values, preserving the final values
// and doing no allocation proportional to an unbounded history.
func Sparkline(values []int, width int) string {
	if width < 1 {
		return ""
	}
	if len(values) > width {
		values = values[len(values)-width:]
	}
	maximum := 0
	for _, value := range values {
		if value > maximum {
			maximum = value
		}
	}
	if maximum == 0 {
		maximum = 1
	}
	var out strings.Builder
	out.Grow(width)
	for _, value := range values {
		if value < 0 {
			value = 0
		}
		index := value * (len(levels) - 1) / maximum
		out.WriteRune(levels[index])
	}
	for index := len(values); index < width; index++ {
		out.WriteRune('·')
	}
	return out.String()
}

// CommitBuckets counts already-loaded commit timestamps into recent time
// buckets. Old and future timestamps are ignored and output is bounded.
func CommitBuckets(unixSeconds []int64, now time.Time, bucket time.Duration, width int) []int {
	if width < 1 {
		return nil
	}
	if width > 32 {
		width = 32
	}
	if bucket <= 0 {
		bucket = 24 * time.Hour
	}
	counts := make([]int, width)
	for _, unix := range unixSeconds {
		at := time.Unix(unix, 0)
		age := now.Sub(at)
		if age < 0 {
			continue
		}
		index := width - 1 - int(age/bucket)
		if index >= 0 && index < width {
			counts[index]++
		}
	}
	return counts
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
