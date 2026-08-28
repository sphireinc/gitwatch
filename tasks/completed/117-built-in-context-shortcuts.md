# Task 117 — Add built-in context-pane shortcuts

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Task 116

## Objective

Make lower context panes discoverable and usable without requiring keymap
configuration.

## Default bindings

- `T` — commit tree.
- `P` — unpushed commits. (`U` already unstages all files in Status.)
- `B` — branch summary.

The current `b` full branches workspace remains available for branch operations.
Bindings must be represented as navigation actions, appear in context help and
the command palette, and be safely overrideable through existing profiles and
direct keymap configuration. Collision and reserved-control validation must
remain enforced.

## Acceptance

Shortcuts work from the status workspace, focus the selected pane, preserve
existing context-specific commands, show active bindings in help, and pass
override/collision/keyboard regression tests.

**Status:** Complete

## Implementation summary

Added built-in `T`, `P`, and `B` bindings, command-palette actions, help text,
and config actions (`commit_tree`, `unpushed`, and `branch_summary`). Existing
`b` branch management and `U` unstage-all behavior remain unchanged. The
bindings continue to use profile/direct keymap validation and override rules.
`T` also enables the commit tree on demand when startup visibility is disabled.
