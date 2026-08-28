package committree

import (
	"strings"
	"testing"
)

func TestParsePreservesFieldsAndDropsSGR(t *testing.T) {
	line := "* \x1b[31mabc123\x1b[0m -\x1b[36m (HEAD -> main)\x1b[0m subject \x1b[32m(2 days ago)\x1b[0m \x1b[1;34m<Author>\x1b[0m"
	segments := Parse(line)
	var plain strings.Builder
	roles := map[Role]bool{}
	for _, segment := range segments {
		plain.WriteString(segment.Text)
		roles[segment.Role] = true
	}
	if strings.Contains(plain.String(), "\x1b") || plain.String() != "* abc123 - (HEAD -> main) subject (2 days ago) <Author>" {
		t.Fatalf("plain = %q", plain.String())
	}
	for _, role := range []Role{RoleHash, RoleDecoration, RoleDate, RoleAuthor} {
		if !roles[role] {
			t.Fatalf("missing role %d in %#v", role, segments)
		}
	}
}

func TestParseDropsUnsafeControlsAndHandlesMalformedEscape(t *testing.T) {
	segments := Parse("ok\x1b]8;;https://evil.example\aunsafe\x1b[999x\x00done")
	var plain strings.Builder
	for _, segment := range segments {
		plain.WriteString(segment.Text)
	}
	if strings.ContainsAny(plain.String(), "\x00\x1b") || plain.String() != "okunsafedone" {
		t.Fatalf("plain = %q", plain.String())
	}
}

func FuzzParseNeverPanics(f *testing.F) {
	f.Add("* abc")
	f.Add("\x1b[31mhostile\x1b[0m")
	f.Fuzz(func(t *testing.T, input string) { _ = Parse(input) })
}
