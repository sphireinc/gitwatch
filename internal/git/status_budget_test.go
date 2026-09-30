package git

import (
	"fmt"
	"testing"
)

// TestParseStatus10KAllocationBudget protects the parser's bounded large-tree
// behavior without imposing a CPU-time budget on different machines.
func TestParseStatus10KAllocationBudget(t *testing.T) {
	for _, size := range []int{10_000, 50_000} {
		t.Run(fmt.Sprintf("%d-entries", size), func(t *testing.T) {
			payload := porcelainStatusPayload(size)
			allocations := testing.AllocsPerRun(1, func() {
				if _, err := ParseStatus(payload); err != nil {
					t.Fatal(err)
				}
			})
			budget := float64(size * 10)
			if allocations > budget {
				t.Fatalf("%d status parse allocations %.0f exceed proportional budget %.0f (10 allocations/entry)", size, allocations, budget)
			}
		})
	}
}
