package plugins

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const pluginHelperModeEnv = "GITWATCH_PLUGIN_RUNTIME_HELPER"

func init() {
	switch os.Getenv(pluginHelperModeEnv) {
	case "parent":
		childStarted := os.Getenv("GITWATCH_PLUGIN_RUNTIME_CHILD_STARTED")
		childSurvived := os.Getenv("GITWATCH_PLUGIN_RUNTIME_CHILD_SURVIVED")
		command := exec.Command(os.Args[0], "--gitwatch-plugin")
		command.Env = replaceTestEnv(os.Environ(), pluginHelperModeEnv, "child")
		command.Env = replaceTestEnv(command.Env, "GITWATCH_PLUGIN_RUNTIME_CHILD_STARTED", childStarted)
		command.Env = replaceTestEnv(command.Env, "GITWATCH_PLUGIN_RUNTIME_CHILD_SURVIVED", childSurvived)
		if err := command.Start(); err != nil || !waitForTestFile(childStarted, 1500*time.Millisecond) {
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv("GITWATCH_PLUGIN_RUNTIME_PARENT_STARTED"), []byte("started"), 0o600); err != nil {
			os.Exit(2)
		}
		for {
			time.Sleep(time.Hour)
		}
	case "child":
		if err := os.WriteFile(os.Getenv("GITWATCH_PLUGIN_RUNTIME_CHILD_STARTED"), []byte("started"), 0o600); err != nil {
			os.Exit(2)
		}
		time.Sleep(4 * time.Second)
		if err := os.WriteFile(os.Getenv("GITWATCH_PLUGIN_RUNTIME_CHILD_SURVIVED"), []byte("survived"), 0o600); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
}

func replaceTestEnv(environment []string, name, value string) []string {
	prefix := name + "="
	filtered := environment[:0]
	for _, entry := range environment {
		if !strings.HasPrefix(entry, prefix) {
			filtered = append(filtered, entry)
		}
	}
	return append(filtered, prefix+value)
}

func waitForTestFile(path string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

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

func TestRuntimeTimeoutKillsPluginProcessTree(t *testing.T) {
	directory := t.TempDir()
	parentStarted := filepath.Join(directory, "parent-started")
	childStarted := filepath.Join(directory, "child-started")
	childSurvived := filepath.Join(directory, "child-survived")
	t.Setenv(pluginHelperModeEnv, "parent")
	t.Setenv("GITWATCH_PLUGIN_RUNTIME_PARENT_STARTED", parentStarted)
	t.Setenv("GITWATCH_PLUGIN_RUNTIME_CHILD_STARTED", childStarted)
	t.Setenv("GITWATCH_PLUGIN_RUNTIME_CHILD_SURVIVED", childSurvived)

	manifest := Manifest{ID: "timeout", Name: "Timeout", Version: "1", APIVersion: APIVersion, Executable: os.Args[0]}
	started := time.Now()
	_, err := (Runtime{Timeout: 2 * time.Second}).Run(context.Background(), manifest, nil)
	if !errors.Is(err, ErrPluginTimeout) {
		t.Fatalf("timed plugin result error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 4*time.Second {
		t.Fatalf("plugin deadline returned too late: %s", elapsed)
	}
	if !waitForTestFile(parentStarted, 100*time.Millisecond) || !waitForTestFile(childStarted, 100*time.Millisecond) {
		t.Fatal("plugin parent or descendant did not start before timeout")
	}
	time.Sleep(2200 * time.Millisecond)
	if _, statErr := os.Stat(childSurvived); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("plugin descendant survived process-tree cancellation: stat error=%v", statErr)
	}
}

func TestHandshakeRejectsUnsupportedManifestBeforeExecution(t *testing.T) {
	manifest := Manifest{ID: "plugin", Name: "Plugin", Version: "1", APIVersion: APIVersion2 + 1, Executable: "missing", Capabilities: []Capability{CapabilityPanel}}
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

func TestHandshakeSupportsAPI2AndHostRenderedCapabilities(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable executable fixture uses a POSIX script")
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "api2-plugin")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"type\":\"handshake\",\"payload\":{\"api_version\":2,\"accepted\":true,\"capabilities\":[\"table\"]}}'\n"
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "api2", Name: "API 2", Version: "2", APIVersion: APIVersion2, Executable: executable, Capabilities: []Capability{CapabilityTable}}
	negotiation, err := (Runtime{}).Handshake(context.Background(), manifest, []Capability{CapabilityTable})
	if err != nil {
		t.Fatal(err)
	}
	if !negotiation.Accepted || negotiation.APIVersion != APIVersion2 || len(negotiation.Capabilities) != 1 || negotiation.Capabilities[0] != CapabilityTable {
		t.Fatalf("API-2 negotiation = %#v", negotiation)
	}
}

func TestHandshakeDegradesUnknownAPI2Capability(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("portable executable fixture uses a POSIX script")
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "future-plugin")
	script := "#!/bin/sh\nprintf '%s\\n' '{\"type\":\"handshake\",\"payload\":{\"api_version\":2,\"accepted\":true,\"capabilities\":[]}}'\n"
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{ID: "future", Name: "Future", Version: "2", APIVersion: APIVersion2, Executable: executable, Capabilities: []Capability{Capability("future_surface")}}
	negotiation, err := (Runtime{}).Handshake(context.Background(), manifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !negotiation.Accepted || negotiation.APIVersion != APIVersion2 || len(negotiation.Capabilities) != 0 {
		t.Fatalf("future capability negotiation = %#v", negotiation)
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
