package plugins

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
