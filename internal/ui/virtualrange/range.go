// Package virtualrange provides bounded logical ranges for terminal lists.
package virtualrange

// Range is a half-open logical item range [Start, End).
type Range struct {
	Start int
	End   int
}

// Compute returns the viewport plus bounded overscan for a logical list.
// Offset is the first logical item at the top of the viewport. All inputs are
// clamped so callers can safely use stale offsets after a refresh or filter.
func Compute(total, offset, viewport, overscan int) Range {
	if total <= 0 {
		return Range{}
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= total {
		offset = total - 1
	}
	if viewport < 1 {
		viewport = 1
	}
	if overscan < 0 {
		overscan = 0
	}
	start := offset - overscan
	if start < 0 {
		start = 0
	}
	end := offset + viewport + overscan
	if end > total {
		end = total
	}
	return Range{Start: start, End: end}
}

// Len returns the number of logical items in the range.
func (r Range) Len() int {
	if r.End <= r.Start {
		return 0
	}
	return r.End - r.Start
}
