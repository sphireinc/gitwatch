package remotes

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	for _, input := range []string{"https://alice:secret@example.com/repo.git", "https://example.com/repo.git?token=secret"} {
		if got := Redact(input); got == input || strings.Contains(got, "secret") {
			t.Fatalf("secret was not redacted: %q", got)
		}
	}
}
