package plugins

import "testing"

func BenchmarkCapabilityNegotiation(b *testing.B) {
	manifest := Manifest{ID: "benchmark", Name: "Benchmark", Version: "1", APIVersion: APIVersion, Executable: "plugin", Capabilities: []Capability{CapabilityCommand, CapabilityPanel, CapabilityRepositoryRead}}
	supported := []Capability{CapabilityCommand, CapabilityPanel, CapabilityStatusWidget, CapabilityRepositoryRead}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := Negotiate(manifest, supported)
		if !result.Accepted {
			b.Fatal("negotiation unexpectedly rejected manifest")
		}
	}
}
