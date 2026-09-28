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

func TestV2MigrationGoldenFixturePreservesExistingSettingsReadOnly(t *testing.T) {
	for _, name := range []string{"GITWATCH_PROFILE", "GITWATCH_THEME", "GITWATCH_MOTION", "GITWATCH_WATCH", "GITWATCH_INTERVAL"} {
		t.Setenv(name, "")
	}
	path := filepath.Join("testdata", "v2", "config.json")
	sourceBefore, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanMigrationFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if plan.SourceVersion != 2 || plan.TargetVersion != 3 || len(plan.Changes) != 3 {
		t.Fatalf("v2 fixture migration plan = %#v", plan)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	gotInspection, err := Inspect(loaded)
	if err != nil {
		t.Fatal(err)
	}
	expectedInspection, err := os.ReadFile(filepath.Join("testdata", "v3", "from-v2-inspect.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotInspection) != strings.TrimSpace(string(expectedInspection)) {
		t.Fatalf("v2 migration inspection differs from golden fixture\n--- got ---\n%s\n--- want ---\n%s", gotInspection, expectedInspection)
	}
	if loaded.Version != 3 || loaded.Theme != "dark" || loaded.Motion != "reduced" || loaded.Watch != "fs" || loaded.Interval != 5*time.Second || loaded.Reconciliation != 45*time.Second || loaded.Debounce != 125*time.Millisecond {
		t.Errorf("v2 scalar settings were not preserved: %#v", loaded)
	}
	if loaded.ShowUntracked || !loaded.ShowIgnored || loaded.Mouse || loaded.Repositories.MaxRepositories != 42 || loaded.Repositories.MaxDepth != 6 {
		t.Errorf("v2 status/repository limits were not preserved: %#v", loaded)
	}
	if len(loaded.Repositories.Groups["clients"]) != 2 || loaded.Repositories.GroupRefresh["clients"] != time.Minute || loaded.Repositories.GroupAutoFetch["clients"] != 2*time.Minute {
		t.Errorf("v2 repository grouping policies were not preserved: %#v", loaded.Repositories)
	}
	if !loaded.Remote.AutoFetch || loaded.Remote.AutoFetchInterval != 15*time.Minute || !loaded.Remote.AutoFetchProfiles["work"].Enabled || loaded.Remote.AutoFetchProfiles["work"].Interval != 10*time.Minute {
		t.Errorf("v2 remote settings were not preserved: %#v", loaded.Remote)
	}
	if !loaded.Plugins.Enabled || len(loaded.Plugins.Directories) != 1 || !loaded.GitHub.Enabled || loaded.GitHub.TokenEnv != "WORK_GITHUB_TOKEN" {
		t.Errorf("v2 provider/plugin settings were not preserved: github=%#v plugins=%#v", loaded.GitHub, loaded.Plugins)
	}
	if loaded.Profile != "work" || loaded.Keymap["quit"] != "x" || loaded.KeymapProfiles["work"]["help"] != "h" {
		t.Errorf("v2 keymap settings were not preserved: %#v", loaded)
	}
	if len(loaded.CustomCommands) != 1 || len(loaded.CustomCommands[0].Prompts) != 1 || loaded.CustomCommands[0].Prompts[0].ID != "ticket" || loaded.CustomCommands[0].Prompts[0].Required != true {
		t.Errorf("v2 custom command prompts were not preserved: %#v", loaded.CustomCommands)
	}
	if loaded.Workspace.PaletteMaxResults != 200 || loaded.Workspace.StatusOverscan != 4 || loaded.Visuals.ActivityBuckets != 8 {
		t.Errorf("v3 typed defaults were not supplied: workspace=%#v visuals=%#v", loaded.Workspace, loaded.Visuals)
	}
	sourceAfter, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(sourceBefore) != string(sourceAfter) {
		t.Fatal("v2 migration modified the golden source fixture")
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
