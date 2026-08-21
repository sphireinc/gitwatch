package patch

import (
	"strconv"
	"strings"
	"testing"
)

func TestLargePatchAllocationBudget(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("diff --git a/large.txt b/large.txt\n@@ -1,10000 +1,10000 @@\n")
	for i := 0; i < 10_000; i++ {
		builder.WriteString("-old-")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString("\n+new-")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteByte('\n')
	}
	data := builder.String()
	allocations := testing.AllocsPerRun(1, func() {
		if _, err := Parse(data); err != nil {
			t.Fatal(err)
		}
	})
	if allocations > 1_000 {
		t.Fatalf("large patch allocations %.0f exceed budget 1000", allocations)
	}
}
