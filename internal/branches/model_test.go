package branches

import "testing"

func TestParse(t *testing.T) {
	v := Parse([]byte("main\x00abc\x00origin/main\x00*\nfeature\x00def\x00\x00 \n"))
	if len(v) != 2 || !v[0].Current || v[1].Remote {
		t.Fatal(v)
	}
}
