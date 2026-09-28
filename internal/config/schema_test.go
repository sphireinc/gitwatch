package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const configurationSchemaV3SHA256 = "485684f7923c83fac85631d69aadfcb27ee8be768b868dfd5e6ed319be375373"

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

func TestDocumentedSchemaV3CoversCustomCommandPrompts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "configuration.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("schema JSON: %v", err)
	}
	var customCommands struct {
		Items struct {
			Properties map[string]json.RawMessage `json:"properties"`
		} `json:"items"`
	}
	if err := json.Unmarshal(schema.Properties["custom_commands"], &customCommands); err != nil {
		t.Fatalf("custom command schema: %v", err)
	}
	var prompts struct {
		Items struct {
			Required   []string                   `json:"required"`
			Properties map[string]json.RawMessage `json:"properties"`
			AllOf      json.RawMessage            `json:"allOf"`
		} `json:"items"`
	}
	if err := json.Unmarshal(customCommands.Items.Properties["prompts"], &prompts); err != nil {
		t.Fatalf("prompt schema: %v", err)
	}
	for _, required := range []string{"id", "label", "kind"} {
		if !slices.Contains(prompts.Items.Required, required) {
			t.Errorf("prompt schema does not require %q", required)
		}
	}
	for _, name := range []string{"id", "label", "kind", "required", "pattern", "min_length", "max_length", "options", "options_source", "default"} {
		if _, ok := prompts.Items.Properties[name]; !ok {
			t.Errorf("prompt schema is missing property %q", name)
		}
	}
	for _, name := range []string{"min_length", "max_length"} {
		var bound struct {
			Type    string `json:"type"`
			Minimum int    `json:"minimum"`
		}
		if err := json.Unmarshal(prompts.Items.Properties[name], &bound); err != nil {
			t.Fatalf("prompt %s schema: %v", name, err)
		}
		if bound.Type != "integer" || bound.Minimum != 0 {
			t.Errorf("prompt %s schema = %#v, want non-negative integer", name, bound)
		}
	}
	var kind struct {
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(prompts.Items.Properties["kind"], &kind); err != nil {
		t.Fatalf("prompt kind schema: %v", err)
	}
	for _, expected := range []string{"text", "secret", "confirm", "select", "multi-select"} {
		if !slices.Contains(kind.Enum, expected) {
			t.Errorf("prompt kind schema is missing %q", expected)
		}
	}
	var restrictions []struct {
		If struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
		} `json:"if"`
		Then struct {
			Properties map[string]struct {
				Maximum *int `json:"maximum"`
			} `json:"properties"`
		} `json:"then"`
	}
	if err := json.Unmarshal(prompts.Items.AllOf, &restrictions); err != nil {
		t.Fatalf("prompt conditional schema: %v", err)
	}
	lengthKindsRestricted := false
	for _, restriction := range restrictions {
		kinds := restriction.If.Properties["kind"].Enum
		if len(kinds) != 3 || !slices.Contains(kinds, "confirm") || !slices.Contains(kinds, "select") || !slices.Contains(kinds, "multi-select") {
			continue
		}
		minMax := restriction.Then.Properties["min_length"].Maximum
		maxMax := restriction.Then.Properties["max_length"].Maximum
		lengthKindsRestricted = minMax != nil && maxMax != nil && *minMax == 0 && *maxMax == 0
		break
	}
	if !lengthKindsRestricted {
		t.Error("prompt schema must restrict non-text/non-secret length bounds to zero")
	}
}
