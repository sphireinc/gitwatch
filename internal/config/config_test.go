package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigPrecedenceAndValidation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(`{"theme":"dark","interval":5000000000}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITWATCH_THEME", "light")
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	c = ApplyCLI(c, "high-contrast", "off", "poll", time.Second)
	if c.Theme != "high-contrast" || c.Motion != "off" || c.Watch != "poll" || c.Interval != time.Second {
		t.Fatal(c)
	}
}

func TestNotificationQuietSettingLoadsAndBuildsModelConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"notifications":{"quiet":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil || !loaded.Notifications.Quiet {
		t.Fatalf("quiet setting = %#v err=%v", loaded.Notifications, err)
	}
}

func TestGroupRefreshPolicyValidation(t *testing.T) {
	c := Defaults()
	c.Repositories.GroupRefresh = map[string]time.Duration{"work": time.Minute}
	if err := Validate(c); err != nil {
		t.Fatal(err)
	}
	c.Repositories.GroupRefresh[" "] = time.Minute
	if err := Validate(c); err == nil {
		t.Fatal("blank group policy was accepted")
	}
}

func TestV2DefaultsAndBindingCollisionValidation(t *testing.T) {
	c := Defaults()
	if c.Version != CurrentVersion || c.Remote.PullStrategy != "ff-only" || c.Layout.FilesPercent != 60 || c.Layout.DetailsPercent != 40 || c.Diff.MaxBytes <= 0 || c.Diff.MaxLines <= 0 {
		t.Fatalf("unexpected defaults: %#v", c)
	}
	c.Keymap = map[string]string{"one": "x", "two": "x"}
	if Validate(c) == nil || len(BindingCollisions(c.Keymap)) != 1 {
		t.Fatal("binding collision was not rejected")
	}
}

func TestLayoutPercentValidation(t *testing.T) {
	c := Defaults()
	c.Layout = LayoutConfig{FilesPercent: 50, DetailsPercent: 50}
	if err := Validate(c); err != nil {
		t.Fatal(err)
	}
	c.Layout.DetailsPercent = 49
	if err := Validate(c); err == nil {
		t.Fatal("layout percentages that do not sum to 100 were accepted")
	}
}

func TestDiffBudgetValidation(t *testing.T) {
	c := Defaults()
	c.Diff.MaxBytes = 0
	if err := Validate(c); err == nil {
		t.Fatal("zero diff byte budget was accepted")
	}
	c = Defaults()
	c.Diff.MaxLines = 0
	if err := Validate(c); err == nil {
		t.Fatal("zero diff line budget was accepted")
	}
}

func TestKeymapProfilesPrecedenceAndValidation(t *testing.T) {
	c := Defaults()
	c.Profile = "writer"
	c.KeymapProfiles = map[string]map[string]string{"writer": {"quit": "x", "help": "h"}}
	c.Keymap = map[string]string{"quit": "q"}
	if err := Validate(c); err != nil {
		t.Fatal(err)
	}
	if got := EffectiveKeymap(c); got["quit"] != "q" || got["help"] != "h" {
		t.Fatalf("effective keymap = %#v", got)
	}
	c.Keymap = map[string]string{"unknown": "x"}
	if err := Validate(c); err == nil || !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("unknown action error = %v", err)
	}
	c = Defaults()
	c.Keymap = map[string]string{"quit": "ctrl+c"}
	if err := Validate(c); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("reserved key error = %v", err)
	}
}

func TestLayoutPercentOverflowNormalizesToEqualPanels(t *testing.T) {
	c := Defaults()
	c.Layout = LayoutConfig{FilesPercent: 70, DetailsPercent: 40}
	normalized, changed := NormalizeLayout(c)
	if !changed || normalized.Layout.FilesPercent != 50 || normalized.Layout.DetailsPercent != 50 {
		t.Fatalf("normalized layout = %#v changed=%v", normalized.Layout, changed)
	}
	if err := Validate(normalized); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLogsLayoutOverflowOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"layout":{"files_percent":70,"details_percent":40}}`), 0600); err != nil {
		t.Fatal(err)
	}
	originalStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = writer
	loaded, loadErr := Load(path)
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	os.Stderr = originalStderr
	message, readErr := io.ReadAll(reader)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if loadErr != nil || loaded.Layout.FilesPercent != 50 || loaded.Layout.DetailsPercent != 50 {
		t.Fatalf("loaded overflow layout = %#v err=%v", loaded.Layout, loadErr)
	}
	if count := strings.Count(string(message), "layout percentages exceed 100"); count != 1 {
		t.Fatalf("overflow warning count = %d, output=%q", count, message)
	}
}

func TestLoadMigratesUnversionedAndV1Configuration(t *testing.T) {
	for _, data := range []string{`{"theme":"dark"}`, `{"version":1,"motion":"reduced"}`} {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		loaded, err := Load(path)
		if err != nil || loaded.Version != CurrentVersion {
			t.Fatalf("migration for %s = %#v, %v", data, loaded, err)
		}
	}
}

func TestLoadRejectsFutureConfigurationVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"version":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("future version was accepted")
	}
}
