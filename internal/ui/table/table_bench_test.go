package table

import (
	"fmt"
	"github.com/jusanchez/gitwatch/internal/repo"
	"testing"
)

func BenchmarkTable10KFilter(b *testing.B) {
	entries := make([]repo.Entry, 10000)
	for i := range entries {
		entries[i] = repo.Entry{Path: repo.Path(fmt.Sprintf("dir/%05d.go", i))}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := New(entries)
		m.SetFilter("9999")
	}
}
