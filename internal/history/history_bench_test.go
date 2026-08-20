package history

import (
	"strconv"
	"strings"
	"testing"
)

func benchmarkLogData(count int) []byte {
	var builder strings.Builder
	for i := 0; i < count; i++ {
		builder.WriteString("sha")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString("\x00sha\x00author\x001700000000\x00\x00\x00\x00subject")
		builder.WriteString(strconv.Itoa(i))
		builder.WriteString("\x1e")
	}
	return []byte(builder.String())
}

func BenchmarkParseLog100K(b *testing.B) {
	data := benchmarkLogData(100_000)
	b.ReportMetric(float64(len(data)), "input-bytes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ParseLog(data)
	}
}

func BenchmarkBuildGraph100K(b *testing.B) {
	commits := make([]Commit, 100_000)
	for i := range commits {
		commits[i].SHA = strconv.Itoa(i)
		if i+1 < len(commits) {
			commits[i].Parents = []string{strconv.Itoa(i + 1)}
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildGraph(commits)
	}
}
