# Task 105 — v2 configuration, plugin API, and migration freeze

**Priority:** P1
**Lane:** v2 preparation
**Dependencies:** Tasks 87, 98, 99, 101, 103, and 104; do not block v1 release

## Objective

Prepare a deliberate v2 compatibility boundary for changes that cannot fit the
v1.x promise. This task defines and validates the boundary; it does not ship
v2 behavior prematurely.

## Required work

- Inventory every public or user-persisted contract: module path, CLI flags,
  exit behavior, config schema, environment variables, keymap action IDs,
  plugin wire messages/capabilities, SDK packages, archive layout, and docs.
- Mark each contract as stable, experimental, deprecated, or explicitly v2.
- Freeze machine-readable fixtures for v1 config and plugin API-1 behavior.
- Design v2 migration rules with dry-run inspection, backup/rollback, unknown
  field handling, and clear errors for unsupported future versions.
- Write compatibility and deprecation policy, including how long v1 plugins,
  configs, and command aliases remain readable.
- Define v2 release gates: clean install, upgrade from v1, downgrade safety,
  package/archive verification, cross-platform native matrix, and announcement
  evidence.

## Non-goals

Do not implement interactive rebase, embedded Git, telemetry, arbitrary
in-process plugins, silent destructive actions, or a v2 breaking change merely
to close this planning task.

## Acceptance criteria

- `docs/v2-release-plan.md` contains a complete contract inventory, migration
  examples, fixture references, and explicit non-goals.
- Automated compatibility fixtures prove v1 config/plugin behavior remains
  readable and unchanged on the v1 branch.
- A dry-run migration can explain every change and leaves source/config data
  untouched until explicit confirmation.
- v2 is not declared releasable until the independent v1 release tasks are
  complete and the migration plan has maintainer approval.

## Verification and documentation

Run schema/fixture tests, clean-install and upgrade simulations, SDK/example
compatibility tests, and documentation link checks. Record decisions in the
task completion summary and update the roadmap only after review.

**Status:** Planned
