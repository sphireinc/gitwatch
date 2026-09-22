package reflog

import "testing"

func FuzzParseNeverPanics(f *testing.F) {
	f.Add([]byte("sha\x00HEAD@{0}\x001700000000\x00actor\x00commit: message\x00"))
	f.Add([]byte("malformed"))
	f.Add([]byte("sha\x00selector\x00-1\x00actor\x00subject\x00"))
	f.Fuzz(func(t *testing.T, input []byte) {
		entries, err := parse(input)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.Timestamp < 0 {
				t.Fatalf("negative timestamp returned: %#v", entry)
			}
		}
	})
}
