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

func TestParseDivergence(t *testing.T) {
	behind, ahead, err := ParseDivergence([]byte("12\t7\n"))
	if err != nil || behind != 12 || ahead != 7 {
		t.Fatalf("divergence = %d/%d err=%v", behind, ahead, err)
	}
	if _, _, err := ParseDivergence([]byte("not counts")); err == nil {
		t.Fatal("invalid divergence was accepted")
	}
}
