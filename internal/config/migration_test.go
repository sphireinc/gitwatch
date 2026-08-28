package config

import (
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(report, "v0 -> v2") || !strings.Contains(report, "source file was written") {
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
