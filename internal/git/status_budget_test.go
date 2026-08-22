package git

import (
	"bytes"
	"fmt"
	"testing"
)

// TestParseStatus10KAllocationBudget protects the parser's bounded large-tree
// behavior without imposing a CPU-time budget on different machines.
func TestParseStatus10KAllocationBudget(t *testing.T) {
	var data bytes.Buffer
	for i := 0; i < 10_000; i++ {
		fmt.Fprintf(&data, "1 .M N... 100644 100644 100644 %040d %040d dir/%05d file.go\x00", i, i, i)
	}
	allocations := testing.AllocsPerRun(1, func() {
		if _, err := ParseStatus(data.Bytes()); err != nil {
			t.Fatal(err)
		}
	})
	if allocations > 100_000 {
		t.Fatalf("10k status parse allocations %.0f exceed budget 100000", allocations)
	}
}
