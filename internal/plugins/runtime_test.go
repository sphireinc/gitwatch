package plugins

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRuntimeRejectsInvalidManifest(t *testing.T) {
	_, err := (Runtime{}).Run(context.Background(), Manifest{}, nil)
	if err == nil {
		t.Fatal("invalid manifest was executed")
	}
}

func TestRuntimeRejectsUngrantCapabilityBeforeExecution(t *testing.T) {
	manifest := Manifest{ID: "plugin", Name: "Plugin", Version: "1", APIVersion: APIVersion, Executable: "missing", Capabilities: []Capability{CapabilityPanel}}
	_, err := (Runtime{}).RunWithCapabilities(context.Background(), manifest, nil, nil)
	if !errors.Is(err, ErrCapabilityDenied) {
		t.Fatalf("capability denial = %v", err)
	}
}

func TestLimitedWriterStopsOversizedOutput(t *testing.T) {
	writer := &limitedWriter{writer: discardWriter{}, limit: 3}
	if _, err := writer.Write([]byte("1234")); !errors.Is(err, ErrOutputLimit) {
		t.Fatalf("expected output limit, got %v", err)
	}
}

func TestRuntimeContainsHostilePluginOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable executable fixture uses a POSIX script")
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "hostile-plugin")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nprintf '1234567890'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "hostile", Name: "Hostile", Version: "1", APIVersion: APIVersion, Executable: executable}
	result, err := (Runtime{OutputLimit: 4}).Run(context.Background(), manifest, nil)
	if err == nil || len(result.Stdout) > 4 {
		t.Fatalf("hostile output result = len=%d err=%v", len(result.Stdout), err)
	}
}

func TestHandshakeRejectsUnsupportedManifestBeforeExecution(t *testing.T) {
	manifest := Manifest{ID: "plugin", Name: "Plugin", Version: "1", APIVersion: APIVersion + 1, Executable: "missing", Capabilities: []Capability{CapabilityPanel}}
	result, err := (Runtime{}).Handshake(context.Background(), manifest, nil)
	if err != nil || result.Accepted {
		t.Fatalf("handshake negotiation = %#v, %v", result, err)
	}
}

type discardWriter struct{}

func (discardWriter) Write(data []byte) (int, error) { return len(data), nil }
