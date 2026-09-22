package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const configurationSchemaV3SHA256 = "fcca5988dc5393782cbbf23be342c007db822df1d7c2c27a44a5ee85609545f2"

func TestDocumentedSchemaV3CoversAdvancedConfigurationSurface(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "configuration.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != configurationSchemaV3SHA256 {
		t.Fatalf("configuration schema golden hash = %s, want %s", got, configurationSchemaV3SHA256)
	}
	var schema struct {
		ID         string                     `json:"$id"`
		Title      string                     `json:"title"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("schema JSON: %v", err)
	}
	if schema.ID == "" || schema.Title != "gitwatch configuration v3" {
		t.Fatalf("schema identity = id:%q title:%q", schema.ID, schema.Title)
	}

	for _, name := range []string{
		"version", "repositories", "remote", "github", "plugins", "tools",
		"custom_commands", "workspace", "visuals", "keymap", "keymap_profiles",
	} {
		if _, ok := schema.Properties[name]; !ok {
			t.Errorf("schema is missing top-level property %q", name)
		}
	}

	var remote struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schema.Properties["remote"], &remote); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"auto_fetch", "auto_fetch_interval", "auto_fetch_jitter", "auto_fetch_backoff", "auto_fetch_backoff_max", "auto_fetch_profiles"} {
		if _, ok := remote.Properties[name]; !ok {
			t.Errorf("remote schema is missing property %q", name)
		}
	}

	var workspace struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schema.Properties["workspace"], &workspace); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"palette_max_results", "status_overscan"} {
		if _, ok := workspace.Properties[name]; !ok {
			t.Errorf("workspace schema is missing property %q", name)
		}
	}

	// Inspection may preserve the name of the environment variable used to
	// obtain a token, but the schema must never define a token/password value.
	for name := range schema.Properties {
		if name == "token" || name == "password" || name == "secret" || name == "credential" {
			t.Errorf("schema exposes secret-like top-level property %q", name)
		}
	}
}
