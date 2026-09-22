package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	publicplugin "github.com/sphireinc/git-watch/pkg/plugin"
)

func TestDiscoverBoundsAndSkipsSymlinks(t *testing.T) {
	directory := t.TempDir()
	pluginDir := filepath.Join(directory, "one")
	if err := os.MkdirAll(pluginDir, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id":"one","name":"One","version":"1","api_version":1,"executable":"one"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, err := Discover(context.Background(), []string{directory}, 1)
	if err != nil || len(entries) != 1 || !entries[0].Healthy {
		t.Fatalf("entries = %#v, err=%v", entries, err)
	}
	updated := SetEnabled(entries, "one", false)
	if updated[0].Enabled || !entries[0].Enabled {
		t.Fatal("set enabled mutated source or failed")
	}
}

func TestDecodeContributionsBoundsAndSkipsInvalidMessages(t *testing.T) {
	valid, err := publicplugin.NewContribution("health", publicplugin.Contribution{SchemaVersion: publicplugin.APIVersion2, Kind: "table", Title: "Health", ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := publicplugin.Encode(valid)
	if err != nil {
		t.Fatal(err)
	}
	data := append([]byte(`{"type":"unknown","payload":{}}\n`), encoded...)
	data = append(data, encoded...)
	contributions, err := DecodeContributions(data, 1)
	if err != nil || len(contributions) != 1 || contributions[0].Title != "Health" {
		t.Fatalf("contributions = %#v, err=%v", contributions, err)
	}
	if _, err := DecodeContributions([]byte(fmt.Sprintf("%s\n", string(make([]byte, publicplugin.MaxMessageBytes+1)))), 1); err == nil {
		t.Fatal("oversized scanner input was accepted")
	}
}

func TestProbeRecordsHostRenderedContributionOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable executable fixture uses a POSIX script")
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "probe-plugin")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"type\":\"handshake\",\"payload\":{\"api_version\":2,\"accepted\":true,\"capabilities\":[\"table\"]}}' '{\"type\":\"contribution\",\"id\":\"health\",\"payload\":{\"schema_version\":2,\"kind\":\"table\",\"title\":\"Health\",\"read_only\":true}}'\n"
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	entry := Entry{Manifest: Manifest{ID: "probe", Name: "Probe", Version: "2", APIVersion: APIVersion2, Executable: executable, Capabilities: []Capability{CapabilityTable}}, Enabled: true, Healthy: true}
	entry = Probe(context.Background(), Runtime{}, entry, []Capability{CapabilityTable})
	if !entry.Healthy || len(entry.Contributions) != 1 || entry.Contributions[0].Title != "Health" {
		t.Fatalf("probed entry = %#v", entry)
	}
}

func TestPluginStateRoundTripIsPrivateAndImmutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "plugins.json")
	entries := []Entry{{Manifest: Manifest{ID: "one"}, Enabled: false}, {Manifest: Manifest{ID: "two"}, Enabled: true}}
	if err := SaveState(path, entries); err != nil {
		t.Fatal(err)
	}
	state, err := LoadState(path)
	if err != nil || !state["two"] || state["one"] {
		t.Fatalf("state = %#v err=%v", state, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("state mode = %v", info.Mode().Perm())
	}
	updated := ApplyState([]Entry{{Manifest: Manifest{ID: "one"}, Enabled: true}}, state)
	if updated[0].Enabled {
		t.Fatal("apply state did not disable plugin")
	}
	if entries[0].Enabled {
		t.Fatal("state application mutated source")
	}
}
