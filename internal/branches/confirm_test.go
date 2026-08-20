package branches

import "testing"

func TestDeleteConfirmation(t *testing.T) {
	c := DeletePrompt("feature", true)
	if c.Accept("yes") || !c.Accept("feature") {
		t.Fatal(c.Text())
	}
}
