package app

import "testing"

func TestRestoreRequiresExactConfirmation(t *testing.T) {
	c := RestoreConfirmation("unsafe path", true, true)
	if !c.Accept("yes") || c.Accept("y") || c.Scope != "staged and working-tree content" {
		t.Fatal(c)
	}
}
