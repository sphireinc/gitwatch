package plugins

import (
	"context"
	"os"
	"path/filepath"
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
