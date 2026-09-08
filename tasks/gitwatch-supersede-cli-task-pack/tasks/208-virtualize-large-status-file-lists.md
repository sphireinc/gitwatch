# Task 208: Virtualize large status file lists

**Phase:** Performance and status UX
**Depends on:** 183

## User-reported problem

A repository with 14,953 untracked files makes status-list scrolling slow enough
to feel unusable. The current status model must remain complete and
authoritative, but the TUI should not construct, style, lay out, and render
every file row on every frame.

## Goal

Introduce a tested, terminal-native visible-range/overscan presentation layer
for the status file list so very large untracked, staged, modified, renamed,
and conflicted sets remain responsive while preserving exact selection,
filtering, diff, staging, restore, and accessibility behavior.

The design may take conceptual inspiration from:

- TanStack Virtual: headless, framework-agnostic virtualization with visible
  ranges, overscan, measured/dynamic sizing, and controlled scrolling:
  https://github.com/TanStack/virtual
- virtua: small virtual-list primitives, dynamic-size handling, scroll-position
  adjustment, imperative scrolling, keyboard navigation, and placeholders:
  https://github.com/inokawa/virtua

These are design references, not dependencies. gitwatch is a Go/Bubble Tea
terminal application and must implement the appropriate terminal-native
behavior without introducing a JavaScript runtime or web UI dependency.

## Non-negotiable constraints

- The authoritative repository snapshot remains the complete parsed
  `git status --porcelain=v2 -z --branch --untracked-files=all` result. Never
  truncate Git state merely to make rendering faster.
- Virtualization applies to presentation work, not to Git parsing, selection
  semantics, filtering correctness, mutation targeting, or diff lookup.
- Preserve byte-exact paths and existing path safety. Spaces, tabs, unicode,
  quotes, leading hyphens, renames, conflicts, and unusual paths must retain
  their current behavior.
- Keyboard and mouse interactions must address the same logical row. Page,
  home/end, selection restoration after refresh/filter changes, and diff/stage/
  restore actions must not depend on whether a row is currently rendered.
- Do not perform Git, filesystem, or expensive measurement work in the render
  path. Watcher events remain refresh hints, and refresh cancellation/lifecycle
  behavior must remain unchanged.
- Keep 80x24, `NO_COLOR`, reduced-motion, and screen-reader/terminal escape
  safety behavior intact. Do not use animation to hide missed frames.
- Do not introduce an unbounded cache, goroutine, subprocess, or retained
  styled-string copy for every visible-range update.

## Implementation steps

1. Profile the current status rendering and scrolling path with a deterministic
   repository fixture containing at least 14,953 untracked files, identifying
   allocations and work performed per frame, per scroll event, and per refresh.
2. Define a pure visible-range abstraction with total item count, viewport
   height, scroll offset, overscan, row extent policy, and logical selected
   index. Add invariants for clamping, page movement, home/end, refresh, and
   filtering.
3. Integrate the abstraction with the status file model so only the visible
   range plus bounded overscan is transformed into terminal rows. Preserve a
   stable logical key/path for selection and mutation lookup.
4. Handle variable-height rows deliberately. Either establish and document a
   fixed row contract for status files or add bounded row measurement and
   scroll-anchor correction; do not silently assume one line if details or
   accessibility text can expand a row.
5. Keep diff/context panes and status summaries responsive while the file list
   contains tens of thousands of entries. Ensure filtering and sorting operate
   on the complete logical set but do not rebuild unrelated rendered rows.
6. Add keyboard and mouse parity tests, including fast repeated scrolling,
   selection changes outside the rendered range, refresh while scrolled, and
   filter changes that remove the selected path.
7. Add benchmark and regression gates for 14,953 and a larger stress fixture.
   Record allocations, visible-row work, scroll latency, and refresh-to-first-
   usable-frame behavior on supported development platforms. Establish
   thresholds from a baseline rather than hiding regressions behind a single
   average benchmark.
8. Document the model and tuning knobs for maintainers, including overscan,
   row extent, viewport changes, and why the complete authoritative snapshot
   remains in memory.

## Expected code areas

- `internal/app/status_view.go`
- `internal/app/app.go` status selection/scroll handling
- `internal/ui/` or a new pure visible-range package
- status/table model tests and deterministic large-repository fixtures
- performance benchmarks and maintainer documentation

## Verification

- A deterministic fixture with at least 14,953 untracked files.
- Focused unit tests for visible-range arithmetic and selection invariants.
- App tests for scroll, page movement, filtering, refresh, diff, stage, and
  restore routing with rows outside the rendered range.
- Benchmark comparison showing bounded per-frame rendered-row work and a
  materially lower scroll cost than the pre-virtualization implementation.
- `go test ./...`, `go test -race ./...`, `go vet ./...`, formatter/lint gates,
  `git diff --check`, and the repository performance gate.
- Manual 80x24 terminal acceptance with `NO_COLOR`, reduced motion, mouse
  navigation, and a 14,953-file repository.

## Acceptance criteria

- [ ] The complete authoritative status snapshot remains correct for all
  14,953+ entries.
- [ ] Status rendering and scrolling process only a bounded visible range plus
  documented overscan rather than materializing every row per frame.
- [ ] Selection, filtering, diff, stage, restore, page movement, and refresh
  remain logically correct for off-screen rows.
- [ ] No visible regression occurs at 80x24, with `NO_COLOR`, or with reduced
  motion; keyboard and mouse reach equivalent rows/actions.
- [ ] Benchmarks and manual acceptance demonstrate that the reported large
  repository no longer scrolls at a crawl.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] Large-repository benchmark and regression evidence recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format/performance evidence recorded.
- [ ] Native/manual terminal evidence recorded.
- [ ] Known limitations/deferred work documented.
