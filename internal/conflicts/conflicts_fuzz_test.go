package conflicts

import "testing"

func FuzzParseIndexNeverPanicsOrExceedsInputBound(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte("100644 base 1\tpath\x00100644 ours 2\tpath\x00100644 theirs 3\tpath\x00"),
		[]byte("malformed"),
		[]byte("100644 oid 4\tpath\x00"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		conflicts, err := ParseIndex(input)
		if err != nil {
			return
		}
		for _, conflict := range conflicts {
			if len(conflict.Path) == 0 {
				t.Fatal("parser returned conflict with empty path")
			}
		}
	})
}
