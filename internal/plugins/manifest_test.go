package plugins

import "testing"

func TestManifestValidationAndNegotiation(t *testing.T) {
	manifest, err := DecodeManifest([]byte(`{"id":"github.prs","name":"PRs","version":"1.0.0","api_version":1,"executable":"gitwatch-plugin-prs","capabilities":["command","panel"]}`))
	if err != nil {
		t.Fatal(err)
	}
	result := Negotiate(manifest, []Capability{CapabilityPanel})
	if !result.Accepted || len(result.Capabilities) != 1 || result.Capabilities[0] != CapabilityPanel {
		t.Fatalf("unexpected negotiation: %#v", result)
	}
}

func TestManifestRejectsInvalidAPIAndDuplicateCapabilities(t *testing.T) {
	manifest := Manifest{ID: "plugin", Name: "Plugin", Version: "1.0.0", APIVersion: APIVersion, Executable: "plugin", Capabilities: []Capability{CapabilityPanel, CapabilityPanel}}
	if manifest.Validate() == nil {
		t.Fatal("duplicate capabilities were accepted")
	}
	manifest.Capabilities = nil
	manifest.APIVersion++
	if manifest.Validate() == nil {
		t.Fatal("unsupported API was accepted")
	}
}
