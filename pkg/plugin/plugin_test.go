package plugin

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWireCompatibility(t *testing.T) {
	want := Message{Type: "status", ID: "1", Payload: []byte(`{"text":"ready"}`)}
	data, err := Encode(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil || got.Type != want.Type || string(got.Payload) != string(want.Payload) {
		t.Fatalf("unexpected message: %#v, %v", got, err)
	}
}

func TestDecodeRejectsHostileMessageSizes(t *testing.T) {
	if _, err := Decode([]byte("{\"type\":\"status\"}\n{\"type\":\"extra\"}")); err == nil {
		t.Fatal("multiple JSON messages were accepted")
	}
	if _, err := Decode([]byte(`{"type":"` + string(make([]byte, MaxFieldBytes+1)) + `"}`)); err == nil {
		t.Fatal("oversized type field was accepted")
	}
	if _, err := Decode(bytes.Repeat([]byte{'x'}, MaxMessageBytes+1)); err == nil {
		t.Fatal("oversized message was accepted")
	}
}

func TestManifestAndNegotiationDefineStableContract(t *testing.T) {
	manifest := Manifest{ID: "demo", Name: "Demo", Version: "1.0.0", APIVersion: APIVersion, Capabilities: []Capability{Panel, StatusWidget}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	response := Negotiate(APIVersion, manifest.Capabilities, []Capability{Panel, StatusWidget})
	if !response.Accepted || len(response.Capabilities) != 2 {
		t.Fatalf("negotiation = %#v", response)
	}
	if Negotiate(APIVersion, []Capability{Command}, nil).Accepted {
		t.Fatal("unsupported capability was accepted")
	}
	unknown := Manifest{ID: "demo", Name: "Demo", Version: "1", APIVersion: APIVersion, Capabilities: []Capability{Capability("unknown")}}
	if unknown.Validate() == nil {
		t.Fatal("unknown capability was accepted")
	}
}

func TestV1WireFixturesRemainDecodable(t *testing.T) {
	fixtures, err := filepath.Glob(filepath.Join("testdata", "v1", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 3 {
		t.Fatalf("got %d v1 fixtures, want 3", len(fixtures))
	}
	for _, fixture := range fixtures {
		data, err := os.ReadFile(fixture)
		if err != nil {
			t.Fatal(err)
		}
		message, err := Decode(data)
		if err != nil {
			t.Fatalf("%s: %v", fixture, err)
		}
		if message.Type != MessageHandshake {
			continue
		}
		var payload struct {
			APIVersion int `json:"api_version"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			t.Fatalf("%s payload: %v", fixture, err)
		}
		if payload.APIVersion != APIVersion {
			t.Fatalf("%s: got API version %d, want %d", fixture, payload.APIVersion, APIVersion)
		}
	}
}
