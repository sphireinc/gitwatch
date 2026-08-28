# v2 release plan

This is a future compatibility-boundary plan, not a gate for the first public
v1 release. Configuration schema version 2 and plugin API version 1 are
independent contract numbers and may ship in gitwatch v1.0.0.

The v2 configuration schema is currently version `2`; unknown future
configuration versions are rejected. The plugin wire contract is currently
API version `1`, and its compatibility fixtures live under
`pkg/plugin/testdata/v1/`. These are the freeze points for the next release.

## Contract inventory

| Contract | Current boundary | Classification | v2 rule |
|---|---|---|---|
| Go module and public SDK import | `github.com/sphireinc/git-watch/pkg/plugin` | stable API-1 | Incompatible wire changes require API-2 and new fixtures |
| CLI flags and exit behavior | config, inspection, diagnostics, and release flags | stable | Preserve existing flags and exit classes; document breaking changes |
| Configuration file | JSON schema version 2 | stable v1 contract | Read v1/unversioned files in memory; reject future versions; never silently rewrite |
| Environment variables | config/profile and theme/motion/watch/provider overrides | stable | Preserve precedence and redact values from diagnostics |
| Keymap action IDs | default, profile, and direct keymap actions | stable | Add actions only; renames/removals require aliases and deprecation |
| Plugin wire messages | newline-delimited JSON, API version 1 | stable API-1 | New required semantics require API-2 negotiation and fixtures |
| Plugin capabilities and manifests | command, panel, and status-widget permissions | stable API-1 | Capability meaning cannot narrow silently |
| Archive layout and metadata | reproducible archives, checksums, SBOM, provenance | stable release contract | New formats are additive; existing archive names remain readable |
| User-visible docs and bindings | README, UX, keymap, and configuration docs | stable guidance | Deprecations identify the last supported release and replacement |

## Migration behavior

`gitwatch --config-migration-dry-run --config <path>` reports the source and
target schema versions plus every in-memory change. It never writes the source
file. Normal startup applies the same migration in memory; future configuration
versions are rejected before startup. A future write mode must create a backup,
show the dry-run plan, require confirmation, write atomically, and retain
rollback instructions.

An unversioned or version-1 file is reported as `v0 -> v2` or `v1 -> v2`, with
current defaults supplied for fields absent from the source. Version 2 files
report that no migration is required. Version 3 or newer files fail closed and
remain untouched.

## Fixtures and release gates

`internal/config/migration_test.go` covers unversioned/version-1 in-memory
migration and future-version rejection. The plugin API-1 request, response,
and status-widget fixtures under `pkg/plugin/testdata/v1/` are loaded through
the public SDK by `pkg/plugin/plugin_test.go` and are the immutable v1 wire
fixtures.

The v2 gate is schema and fixture tests, full tests/race/vet/format/lint/
security checks, clean-install and upgrade simulations, archive/checksum/SBOM/
provenance verification, native cross-platform validation, and maintainer
review of this inventory. Downgrade safety means a v2 tool never rewrites a v1
source file and an unsupported future file remains recoverable.

Before a v2 tag, maintainers must:

- review the schema and plugin fixtures as a compatibility change;
- update `CHANGELOG.md`, migration notes, and the version package;
- run `./scripts/release-check.sh` and `./scripts/verify-release-artifacts.sh`;
- perform clean-install and upgrade checks from the previous release;
- publish the signed tag, archives, SDK documentation, and announcement.

This plan intentionally records publication and announcement as external
release actions; passing local tests is not evidence that they occurred.
