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

func TestV2DefaultsAndBindingCollisionValidation(t *testing.T) {
	c := Defaults()
	if c.Version != 2 || c.Remote.PullStrategy != "ff-only" {
		t.Fatalf("unexpected defaults: %#v", c)
	}
	c.Keymap = map[string]string{"one": "x", "two": "x"}
	if Validate(c) == nil || len(BindingCollisions(c.Keymap)) != 1 {
		t.Fatal("binding collision was not rejected")
	}
}
