package sync

import "testing"

func FuzzParseArchive(f *testing.F) {
	f.Add([]byte("not an archive"))
	f.Fuzz(func(_ *testing.T, input []byte) {
		_, _, _ = ParseArchive(input, Config{Commit: "0123456789012345678901234567890123456789", MaxArchiveBytes: 1 << 20, MaxEntryBytes: 1 << 16})
	})
}
