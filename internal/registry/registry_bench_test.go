package registry

import (
	"strconv"
	"testing"
)

func BenchmarkRows1000Repositories(b *testing.B) {
	results := make([]StatusResult, 1_000)
	for i := range results {
		results[i].Repository = Repository{Path: "/repo/" + strconv.Itoa(i), Name: "repo-" + strconv.Itoa(i)}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Rows(results)
	}
}
