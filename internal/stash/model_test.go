package stash

import "testing"

func TestParse(t *testing.T) {
	v := Parse([]byte("stash@{0} abc 1700000000 On main: save work\n"))
	if len(v) != 1 || v[0].Ref != "stash@{0}" || v[0].Unix != 1700000000 {
		t.Fatal(v)
	}
}
