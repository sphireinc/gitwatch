package platform

import "testing"

func TestSafeText(t *testing.T) {
	got := SafeText("ok\x1b[31m;$(touch pwned)\nnext")
	if got == "" || got[0] != 'o' || got == "ok\x1b[31m;$(touch pwned)\nnext" {
		t.Fatalf("unsafe text: %q", got)
	}
}
