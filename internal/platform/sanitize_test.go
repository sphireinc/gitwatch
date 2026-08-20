package platform

import (
	"strings"
	"testing"
)

func TestSafeText(t *testing.T) {
	got := SafeText("ok\x1b[31m;$(touch pwned)\nnext")
	if got == "" || got[0] != 'o' || got == "ok\x1b[31m;$(touch pwned)\nnext" {
		t.Fatalf("unsafe text: %q", got)
	}
}

func TestRedactSecrets(t *testing.T) {
	got := RedactSecrets("token=abc123 password:secret https://user:pass@example.com/repo")
	if strings.Contains(got, "abc123") || strings.Contains(got, "secret") || strings.Contains(got, "user:pass") {
		t.Fatalf("secret leaked: %q", got)
	}
}
