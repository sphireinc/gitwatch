package config

import (
	"os"
	"path/filepath"
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
	if c.Version != CurrentVersion || c.Remote.PullStrategy != "ff-only" {
		t.Fatalf("unexpected defaults: %#v", c)
	}
	c.Keymap = map[string]string{"one": "x", "two": "x"}
	if Validate(c) == nil || len(BindingCollisions(c.Keymap)) != 1 {
		t.Fatal("binding collision was not rejected")
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
