package blame

import "testing"

func FuzzParseNeverPanics(f *testing.F) {
	f.Add([]byte("1234567890abcdef 1 1 1\nauthor Ada\n\tcontent\n"))
	f.Add([]byte("not a blame record\n\x00\xff"))
	f.Fuzz(func(t *testing.T, input []byte) {
		lines := Parse(input)
		for _, line := range lines {
			if line.NumLines < 1 {
				t.Fatalf("parser returned invalid line count: %#v", line)
			}
		}
	})
}
