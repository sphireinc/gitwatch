package branches

import "testing"

func TestParseTracking(t *testing.T) {
	for value, want := range map[string][2]int{">": {1, 0}, "<": {0, 1}, "<>": {1, 1}, "=": {0, 0}} {
		ahead, behind := ParseTracking(value)
		if ahead != want[0] || behind != want[1] {
			t.Fatalf("%q: got %d/%d", value, ahead, behind)
		}
	}
}
