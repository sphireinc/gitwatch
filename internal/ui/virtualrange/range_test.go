package virtualrange

import "testing"

func TestComputeClampsAndAddsOverscan(t *testing.T) {
	tests := []struct {
		name                string
		total, offset, view int
		overscan            int
		want                Range
	}{
		{name: "middle", total: 100, offset: 40, view: 10, overscan: 2, want: Range{Start: 38, End: 52}},
		{name: "top", total: 100, offset: 0, view: 10, overscan: 3, want: Range{Start: 0, End: 13}},
		{name: "bottom", total: 100, offset: 99, view: 10, overscan: 3, want: Range{Start: 96, End: 100}},
		{name: "stale offset", total: 4, offset: 20, view: 2, overscan: 1, want: Range{Start: 2, End: 4}},
		{name: "empty", total: 0, offset: 2, view: 4, overscan: 1, want: Range{}},
		{name: "negative inputs", total: 4, offset: -2, view: 0, overscan: -1, want: Range{Start: 0, End: 1}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Compute(test.total, test.offset, test.view, test.overscan); got != test.want {
				t.Fatalf("Compute() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRangeLenNeverNegative(t *testing.T) {
	for _, test := range []Range{{}, {Start: 3, End: 2}, {Start: 2, End: 5}} {
		if got := test.Len(); got < 0 {
			t.Fatalf("Len(%#v) = %d", test, got)
		}
	}
}
