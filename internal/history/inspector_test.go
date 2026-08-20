package history

import "testing"

func TestParseStats(t *testing.T) {
	stats := parseStats([]byte("3\t1\tinternal/app/app.go\n-\t-\tassets/logo.bin\n"))
	if len(stats) != 2 || stats[0].Added != 3 || stats[0].Deleted != 1 || !stats[1].Binary {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if got := statPaths([]byte("3\t1\tfile.txt\n")); len(got) != 1 || got[0] != "file.txt" {
		t.Fatalf("unexpected paths: %#v", got)
	}
}
