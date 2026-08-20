package platform

import "testing"

func TestOpenURLCommandRejectsUnsafeSchemes(t *testing.T) {
	if _, err := OpenURLCommand("javascript:alert(1)"); err == nil {
		t.Fatal("unsafe URL accepted")
	}
	if _, err := OpenURLCommand("https://github.com/octo/repo/pull/1"); err != nil {
		t.Fatal(err)
	}
}
