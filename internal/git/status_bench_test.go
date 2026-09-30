package git

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkParseStatus10K(b *testing.B) {
	benchmarkParseStatusSize(b, 10_000)
}

func BenchmarkParseStatusScale(b *testing.B) {
	for _, size := range []int{10_000, 50_000} {
		b.Run(fmt.Sprintf("%d-entries", size), func(b *testing.B) {
			benchmarkParseStatusSize(b, size)
		})
	}
}

func benchmarkParseStatusSize(b *testing.B, size int) {
	b.Helper()
	payload := porcelainStatusPayload(size)
	b.ReportAllocs()
	b.ReportMetric(float64(len(payload)), "bytes")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseStatus(payload); err != nil {
			b.Fatal(err)
		}
	}
}

func porcelainStatusPayload(size int) []byte {
	var data bytes.Buffer
	for i := 0; i < size; i++ {
		fmt.Fprintf(&data, "1 .M N... 100644 100644 100644 %040d %040d dir/%05d file.go\x00", i, i, i)
	}
	return data.Bytes()
}
