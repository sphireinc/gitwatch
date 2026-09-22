package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanMigrationIsReadOnlyAndExplainsLegacyDefaults(t *testing.T) {
	data := []byte(`{"theme":"dark"}`)
	plan, err := PlanMigration(data)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceVersion != 0 || plan.TargetVersion != CurrentVersion || len(plan.Changes) != 2 {
		t.Fatalf("plan = %#v", plan)
	}
	report := FormatMigrationPlan(plan)
	if !strings.Contains(report, "v0 -> v3") || !strings.Contains(report, "source file was written") {
		t.Fatalf("report = %q", report)
	}
	if string(data) != `{"theme":"dark"}` {
		t.Fatal("migration changed source data")
	}
}

func TestPlanMigrationRejectsFutureVersion(t *testing.T) {
	if _, err := PlanMigration([]byte(`{"version":99}`)); err == nil {
		t.Fatal("future version was accepted")
	}
}

func TestFormatMigrationPlanForCurrentVersion(t *testing.T) {
	report := FormatMigrationPlan(MigrationPlan{SourceVersion: CurrentVersion, TargetVersion: CurrentVersion})
	if !strings.Contains(report, "no migration required") || strings.Contains(report, "source file was written") {
		t.Fatalf("report = %q", report)
	}
}

func TestV2MigrationAddsTypedV3DefaultsWithoutDisablingWatchers(t *testing.T) {
	data := []byte(`{"version":2,"watch":"fs","interval":5000000000,"repositories":{"max_repositories":50},"plugins":{"enabled":true}}`)
	plan, err := PlanMigration(data)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceVersion != 2 || plan.TargetVersion != 3 || len(plan.Changes) != 3 || !strings.Contains(FormatMigrationPlan(plan), "workspace, visuals") {
		t.Fatalf("v2 migration plan = %#v", plan)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 3 || loaded.Watch != "fs" || loaded.Interval != 5*time.Second || !loaded.Plugins.Enabled || loaded.Workspace.PaletteMaxResults != 200 {
		t.Fatalf("v2 loaded config = %#v", loaded)
	}
}

func TestV1ConfigFixtureRemainsReadable(t *testing.T) {
	path := filepath.Join("testdata", "v1", "config.json")
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != CurrentVersion || loaded.Theme != "dark" || loaded.Motion != "reduced" || loaded.Layout.FilesPercent != 60 {
		t.Fatalf("loaded v1 fixture = %#v", loaded)
	}
	plan, err := PlanMigrationFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceVersion != 1 || plan.TargetVersion != CurrentVersion || len(plan.Changes) == 0 {
		t.Fatalf("v1 migration plan = %#v", plan)
	}
}
