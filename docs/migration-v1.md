# Legacy configuration migration

The published gitwatch v1.0.9 release uses configuration schema version 3.
The schema number is independent of the application release number. This guide
covers configuration files created by earlier prerelease and beta builds; it
does not describe an application-version upgrade.

## Configuration migration

Configuration without a version and schema-version-1 or schema-version-2 files
are normalized in memory to schema version 3. gitwatch never rewrites the
source file during startup or migration inspection. Run
`gitwatch --config-migration-dry-run --config <path>` to review the migration
plan, or `gitwatch --config-inspect --config <path>` to inspect the effective
configuration after in-memory defaults are applied.

Schema version 2 introduced repository groups and per-group refresh intervals,
remote pull strategy settings, GitHub/provider settings, plugin directories and
output limits, notification quiet mode, and validated keymap bindings. Schema
version 3 adds bounded workspace and visualization settings. See the
[configuration guide](configuration.md) and [JSON Schema](configuration.schema.json)
for the complete current option list, defaults, and constraints. Future schema
versions are rejected rather than silently rewritten.

## Historical keyboard changes

The core status bindings remain stable. Workbench views are opened with
`b` (branches), `n` (remotes), `v` (repositories), and `E` (plugins when
enabled). In the branch workspace, `/` filters, `s` cycles sorting, and
`c`/`R`/`u`/`N`/`D`/`X` create, rename, set/unset upstream, or delete branches.
Use `?` for the context-sensitive keymap.

## Plugins

Plugins are opt-in, out-of-process programs. Existing plugin manifests remain
API-versioned; third-party code should import only `github.com/sphireinc/git-watch/pkg/plugin`.
Review requested capabilities and enablement after upgrading. Runtime output
limits and capability grants continue to apply after migration.
