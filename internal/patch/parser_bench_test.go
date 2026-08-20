package patch

import (
	"strconv"
	"strings"
	"testing"
)

func BenchmarkParseLargePatch(b *testing.B) {
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
	b.ReportMetric(float64(len(data)), "input-bytes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(data)
	}
}
