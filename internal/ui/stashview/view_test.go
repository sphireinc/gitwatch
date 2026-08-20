package stashview

import (
	"github.com/jusanchez/gitwatch/internal/stash"
	"strings"
	"testing"
)

func TestView(t *testing.T) {
	m := New([]stash.Entry{{Ref: "stash@{0}", Message: "save"}})
	if !strings.Contains(m.View(), "stash@{0}") {
		t.Fatal(m.View())
	}
}
