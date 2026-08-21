package plugins

import (
	"context"
	"errors"
	"io"
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

func TestLimitedWriterReturnsUnderlyingFailure(t *testing.T) {
	want := errors.New("write failed")
	writer := &limitedWriter{writer: failingWriter{err: want}, limit: 3}
	if _, err := writer.Write([]byte("1234")); !errors.Is(err, want) {
		t.Fatalf("expected underlying failure, got %v", err)
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

func TestHandshakeRejectsCapabilitiesNotGrantedByHost(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable executable fixture uses a POSIX script")
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "bad-handshake")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"type\":\"handshake\",\"payload\":{\"api_version\":1,\"accepted\":true,\"capabilities\":[\"panel\"]}}'\n"
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "handshake", Name: "Handshake", Version: "1", APIVersion: APIVersion, Executable: executable, Capabilities: []Capability{CapabilityCommand}}
	_, err := (Runtime{}).Handshake(context.Background(), manifest, []Capability{CapabilityCommand})
	if !errors.Is(err, ErrCapabilityDenied) {
		t.Fatalf("handshake capability validation = %v", err)
	}
}

func TestSuperviseRetriesWithinBoundedPolicy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable executable fixture uses a POSIX script")
	}
	directory := t.TempDir()
	marker := filepath.Join(directory, "attempts")
	executable := filepath.Join(directory, "failing-plugin")
	script := "#!/bin/sh\nprintf x >> '" + marker + "'\nexit 1\n"
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "retry", Name: "Retry", Version: "1", APIVersion: APIVersion, Executable: executable}
	_, err := (Runtime{}).Supervise(context.Background(), manifest, nil, nil, Supervision{MaxRestarts: 2})
	if err == nil {
		t.Fatal("failing plugin unexpectedly succeeded")
	}
	data, readErr := os.ReadFile(marker)
	if readErr != nil || len(data) != 3 {
		t.Fatalf("restart count = %d, err=%v", len(data), readErr)
	}
}

type discardWriter struct{}

func (discardWriter) Write(data []byte) (int, error) { return len(data), nil }

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

var _ io.Writer = failingWriter{}
