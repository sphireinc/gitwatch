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

func TestSDKBuildersRemainAPI1WireCompatible(t *testing.T) {
	message := NewHandshake([]Capability{Command, StatusWidget})
	if message.Type != MessageHandshake {
		t.Fatal(message)
	}
	if _, err := NewCommand("build", CommandSpec{ID: "build", Title: "Build"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPanel("panel", PanelSpec{ID: "panel", Title: "Panel"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewWidget("widget", StatusWidgetSpec{ID: "widget", Text: "ok"}); err != nil {
		t.Fatal(err)
	}
}

func TestVersionedNegotiationDegradesUnknownVNextCapabilities(t *testing.T) {
	response := NegotiateVersions([]int{APIVersion2, APIVersion},
		[]Capability{ContextAction, Capability("future_surface")},
		[]Capability{ContextAction})
	if !response.Accepted || response.APIVersion != APIVersion2 {
		t.Fatalf("versioned negotiation = %#v", response)
	}
	if len(response.Capabilities) != 1 || response.Capabilities[0] != ContextAction {
		t.Fatalf("degraded capabilities = %#v", response.Capabilities)
	}
	if got := NegotiateVersions([]int{99}, nil, nil); got.Accepted {
		t.Fatalf("unsupported version was accepted: %#v", got)
	}
}

func TestContributionIsBoundedDataOnlyAndRejectsControlSequences(t *testing.T) {
	message, err := NewContribution("health", Contribution{
		SchemaVersion: APIVersion2,
		Kind:          "table",
		Title:         "Repository health",
		Columns:       []TableColumn{{ID: "state", Title: "State"}},
		Rows:          []TableRow{{"state": "clean"}},
		ReadOnly:      true,
	})
	if err != nil || message.Type != MessageContribution {
		t.Fatalf("contribution = %#v, %v", message, err)
	}
	if _, err := NewContribution("unsafe", Contribution{SchemaVersion: 1, Kind: "detail", Title: "\x1b[2J"}); err == nil {
		t.Fatal("terminal control sequence was accepted")
	}
	rows := make([]TableRow, MaxContributionRows+1)
	if _, err := NewContribution("large", Contribution{SchemaVersion: 1, Kind: "table", Title: "large", Rows: rows}); err == nil {
		t.Fatal("oversized contribution was accepted")
	}
}

func TestRepositoryMetadataActionUsesOnlyBoundedHostProviderIdentity(t *testing.T) {
	contribution := Contribution{
		SchemaVersion: APIVersion2,
		Kind:          "repository_metadata",
		Title:         "GitHub repository",
		Action: &ActionSpec{
			ID: "open-repository", Title: "Open repository metadata", Context: "repository",
			Provider: ActionProviderGitHubRepository, ReadOnly: true,
		},
		ReadOnly: true,
	}
	if _, err := NewContribution("github", contribution); err != nil {
		t.Fatalf("valid provider action rejected: %v", err)
	}
	unsafe := contribution
	unsafe.Action = &ActionSpec{ID: "open", Title: "Open", Provider: "https://github.example/token", ReadOnly: true}
	if _, err := NewContribution("unsafe", unsafe); err == nil {
		t.Fatal("provider URL/userinfo was accepted as a provider identity")
	}
	unsafe = contribution
	unsafe.ReadOnly = false
	if _, err := NewContribution("mutable", unsafe); err == nil {
		t.Fatal("mutable repository metadata action was accepted")
	}
	unsafe = contribution
	unsafe.Action.ReadOnly = false
	if _, err := NewContribution("mutable-action", unsafe); err == nil {
		t.Fatal("mutating repository metadata action was accepted")
	}
}

func TestAPI2ManifestAndVersionedHandshakeAreAdditive(t *testing.T) {
	manifest := Manifest{ID: "demo", Name: "Demo", Version: "2", APIVersion: APIVersion2, Capabilities: []Capability{TableContribution}}
	if err := manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	message := NewHandshakeVersions([]int{APIVersion2, APIVersion}, []Capability{TableContribution})
	var request HandshakeRequest
	if err := json.Unmarshal(message.Payload, &request); err != nil {
		t.Fatal(err)
	}
	if len(request.Versions) != 2 || request.Versions[0] != APIVersion2 {
		t.Fatalf("handshake versions = %#v", request.Versions)
	}
}

func TestV2ContributionFixtureRemainsDecodable(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "v2", "contribution.json"))
	if err != nil {
		t.Fatal(err)
	}
	message, err := Decode(data)
	if err != nil || message.Type != MessageContribution {
		t.Fatalf("fixture = %#v, %v", message, err)
	}
	var contribution Contribution
	if err := json.Unmarshal(message.Payload, &contribution); err != nil {
		t.Fatal(err)
	}
	if err := contribution.Validate(); err != nil {
		t.Fatal(err)
	}
}
