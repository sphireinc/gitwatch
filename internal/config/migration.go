package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// MigrationChange describes one source-file change that an in-memory
// migration would make. It never contains secret values or file contents.
type MigrationChange struct {
	Field  string
	Before string
	After  string
	Reason string
}

// MigrationPlan is a read-only explanation of configuration migration.
type MigrationPlan struct {
	SourceVersion int
	TargetVersion int
	Changes       []MigrationChange
}

// PlanMigration builds a dry-run migration plan without modifying data.
func PlanMigration(data []byte) (MigrationPlan, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return MigrationPlan{}, fmt.Errorf("invalid configuration: %w", err)
	}
	version := 0
	if value, ok := raw["version"]; ok {
		if err := json.Unmarshal(value, &version); err != nil {
			return MigrationPlan{}, fmt.Errorf("invalid configuration version: %w", err)
		}
	}
	if version > CurrentVersion {
		return MigrationPlan{}, fmt.Errorf("unsupported config version %d", version)
	}
	plan := MigrationPlan{SourceVersion: version, TargetVersion: CurrentVersion}
	if version < CurrentVersion {
		before := "missing"
		if version != 0 {
			before = fmt.Sprint(version)
		}
		plan.Changes = append(plan.Changes, MigrationChange{
			Field: "version", Before: before, After: fmt.Sprint(CurrentVersion),
			Reason: "migrate the configuration envelope to the current schema",
		})
		plan.Changes = append(plan.Changes, MigrationChange{
			Field: "new schema defaults", Before: "not present in source", After: "current defaults",
			Reason: "new fields are supplied in memory and the source file is not rewritten",
		})
	}
	return plan, nil
}

// PlanMigrationFile reads a configuration file and returns its read-only plan.
func PlanMigrationFile(path string) (MigrationPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MigrationPlan{}, err
	}
	return PlanMigration(data)
}

// FormatMigrationPlan renders a stable, value-safe dry-run report.
func FormatMigrationPlan(plan MigrationPlan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "configuration migration: v%d -> v%d\n", plan.SourceVersion, plan.TargetVersion)
	if len(plan.Changes) == 0 {
		b.WriteString("no migration required; source file unchanged\n")
		return b.String()
	}
	for _, change := range plan.Changes {
		fmt.Fprintf(&b, "- %s: %s -> %s (%s)\n", change.Field, change.Before, change.After, change.Reason)
	}
	b.WriteString("dry run only: no source file was written\n")
	return b.String()
}
