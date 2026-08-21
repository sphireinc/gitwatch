package history

import "testing"

func TestLargeHistoryAllocationBudgets(t *testing.T) {
	logData := benchmarkLogData(100_000)
	parseAllocations := testing.AllocsPerRun(1, func() { _ = ParseLog(logData) })
	if parseAllocations > 300_000 {
		t.Fatalf("100k history parse allocations %.0f exceed budget 300000", parseAllocations)
	}
	commits := make([]Commit, 100_000)
	for i := range commits {
		commits[i].SHA = string(rune(i))
		if i+1 < len(commits) {
			commits[i].Parents = []string{string(rune(i + 1))}
		}
	}
	graphAllocations := testing.AllocsPerRun(1, func() { _ = BuildGraph(commits) })
	if graphAllocations > 200_000 {
		t.Fatalf("100k graph allocations %.0f exceed budget 200000", graphAllocations)
	}
}
