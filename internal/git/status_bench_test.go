package git

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkParseStatus10K(b *testing.B) {
	var data bytes.Buffer
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(&data, "1 .M N... 100644 100644 100644 %040d %040d dir/%05d file.go\x00", i, i, i)
	}
	payload := data.Bytes()
	b.ReportMetric(float64(len(payload)), "bytes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseStatus(payload); err != nil {
			b.Fatal(err)
		}
	}
}
