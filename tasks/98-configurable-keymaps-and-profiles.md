# Task 98 — Configurable keymaps and user profiles

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 23, 24, 82, and 93

## Objective

Allow users to customize non-dangerous key bindings and maintain named
configuration profiles without making destructive actions easier to trigger or
breaking existing configuration files.

## Required work

- Define a versioned keymap schema with action IDs separate from display text.
- Preserve current defaults exactly when no override is present.
- Validate unknown actions, invalid key sequences, duplicate bindings, reserved
  terminal/control sequences, and collisions across modal/workspace scopes.
- Support explicit profile selection and inspect the effective merged config;
  define precedence among defaults, profile, config file, environment, and CLI.
- Make help, command palette, mouse-equivalent descriptions, and conflict
  diagnostics derive from the same effective binding registry.
- Require deliberate confirmation for remapping destructive actions and ensure
  users cannot create an accidental unconfirmable restore/delete path.
- Provide reset-to-default and migration behavior with atomic, private writes.

## Acceptance criteria

- Existing configs load unchanged and produce identical default behavior.
- Invalid keymaps fail before the TUI starts with actionable field paths.
- Effective-config inspection shows source/profile and resolved bindings without
  exposing secrets or unrelated private values.
- Tests cover precedence, migration, collisions, modal scope, reserved keys,
  reset behavior, and help/palette parity.

## Verification and documentation

Update the JSON schema, configuration docs, keymap docs, examples, and release
migration notes. Run full tests, schema validation, and native keyboard-only
acceptance on all supported platforms.

**Status:** Planned
