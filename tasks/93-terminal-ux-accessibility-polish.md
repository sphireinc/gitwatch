# Task 93 — Terminal UX spacing, accessibility, and capability polish

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Task 92 for baseline evidence; existing layout, theme, mouse,
and status-view contracts

## Objective

Make the dashboard easier to scan in real terminals without weakening its
density or changing keyboard semantics. Establish a measured visual system for
panel insets, headers, separators, row rhythm, empty states, and narrow layouts.

## Required work

- Define terminal-cell spacing tokens for panel left/top insets, divider
  columns, header separation, activity/footer separation, and minimum content.
- Verify that wide, medium, narrow, and too-small layouts do not clip headers,
  footer help, activity messages, diff headings, or wrapped paths.
- Keep mouse hit regions aligned with every inset, blank row, wrapped row, and
  divider; test stage controls separately from row selection.
- Review `NO_COLOR`, colorless, light, dark, high-contrast, and reduced/off
  motion output using symbols and text that remain meaningful without color.
- Improve clean, loading, empty, error, binary, conflict, and unavailable
  provider/plugin states so none is visually blank or ambiguous.
- Test Unicode display width, combining characters, tabs, control-byte
  sanitization, and long paths in every panel.
- Add a repeatable screenshot/recording checklist, while keeping tests
  render-state based and independent of a particular terminal font.

## Non-goals

Do not replace the TUI with a web UI, add telemetry, make mouse interaction
more powerful than keyboard interaction, or introduce terminal-specific escape
sequences without a capability fallback.

## Acceptance criteria

- A documented spacing contract exists and is applied consistently across all
  status and workspace views.
- Every interactive row remains selectable after wrapping, scrolling, resize,
  and panel inset changes.
- Keyboard-only operation is complete; colorless and high-contrast modes pass
  the same semantic assertions as colored output.
- Native evidence covers at least two terminal emulators per desktop platform
  where available, plus the supported minimum size.
- No existing key binding, configuration field, plugin protocol, or status
  mutation semantic changes without an explicit migration note.

## Verification and documentation

Add focused layout/render/hit-test tests, run full tests and race tests, and
update `UX_SPEC.md`, `docs/operator-*`, README screenshots/GIFs, and the keymap
documentation. Record visual limitations separately from automated evidence.

**Status:** Planned
