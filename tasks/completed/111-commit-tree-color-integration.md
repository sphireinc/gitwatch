# Task 111 — Integrate colorized commit trees with refresh and interaction

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Tasks 109 and 110

## Objective

Integrate semantic colored lines into the existing optional pane without
regressing startup loading, refresh behavior, scrolling, the horizontal
separator, file selection, or the right details/diff panel.

## Requirements

Preserve Task 107's request/generation validation, asynchronous cancellation,
refresh after commits/external refs/repository switches, 100-commit default,
offset clamping, `T` focus, keyboard and mouse scrolling, and all responsive
layouts. Style metadata must not affect mouse coordinates or scroll limits. The
separator remains the first row of the lower left pane. Colorized data must not
enter diagnostics unless sanitized.

## Tests and acceptance

Use disposable repositories and injected runners to cover startup, linear and
merge history, decorated refs, external ref refresh, cancellation, stale result
rejection, repository switching, wrapping, scrolling, separator placement,
unchanged right-panel dimensions, disabled mode, and `NO_COLOR`. Include race,
leak, and shutdown coverage. A repository with commits must show the graph
immediately after enabled startup, with bounded and generation-safe history
work.

**Status:** Complete

## Completion summary

Integrated the colorized Git contract and safe semantic parser into the status
commit-tree path without changing pane geometry, separator placement, scrolling,
mouse coordinates, or right-panel behavior. Existing asynchronous request and
repository-generation guards remain authoritative for startup, on-demand,
refresh, cancellation, and repository switching. Colorless rendering and
on-demand `T` activation are covered by app regression tests.
