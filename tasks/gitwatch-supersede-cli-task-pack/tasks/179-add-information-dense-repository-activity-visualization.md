# Task 179: Add information-dense repository activity visualization

**Phase:** Multi-repository differentiation
**Depends on:** 175

## Goal

Add useful htop-style visuals—change heat, diff magnitude and activity sparklines—without making refresh expensive or decorative.

## Non-negotiable constraints

- Live filesystem-driven status is the product core. Do not replace, subordinate, or pause it except for the minimum repository-lock window required by Git itself.
- Filesystem events are refresh hints, never authoritative state. The authoritative worktree snapshot remains `git status --porcelain=v2 -z --branch --untracked-files=all` parsed into immutable repository state.
- Every successful mutation MUST request an authoritative refresh for the affected repository. Long-running sequencer operations must refresh after every observable state transition.
- Multi-repository support is first-class. New domain models and operations MUST carry repository identity/scope and remain correct while other repositories refresh or run unrelated work.
- Do not create unbounded watchers, goroutines, workers, or Git/provider/plugin processes. Reuse bounded registry/operation infrastructure.
- All Git commands use typed argv execution through the Git boundary. Never interpolate repository data into shell command strings. Use `--` where supported and machine-readable/NUL-delimited output where available.
- Bubble Tea owns UI state. Git/network/filesystem/process work never runs in the render path.
- Repository-controlled text is untrusted terminal input and MUST be sanitized before rendering.
- Destructive/history-rewriting actions require scope-specific confirmation. Keep the prohibition on generic `reset --hard`, raw `--force`, and `clean -fd` shortcuts.
- Keyboard and mouse must reach equivalent functionality. New views must work at 80x24, honor `NO_COLOR`, and support full/reduced/off motion.
- Do not reimplement Git. Use Git as source of truth and build safe typed control/presentation layers around it.
- Breaking config/plugin changes require versioning, migration, and compatibility fixtures.

## Implementation steps

1. Derive current change heat/diff bars from status/diff-stat data already available; do not run extra expensive Git commands solely for animation.
2. Add bounded recent-commit activity sparkline from history cache.
3. Always pair color with text/icon semantics and provide NO_COLOR/ASCII fallback.
4. Respect reduced/off motion; data visuals need not animate.
5. Clip/virtualize to terminal width and use display-width-aware glyph handling.
6. Make every visual optional/configurable if it consumes meaningful space.

## Verification

- NO_COLOR/ASCII, 80x24, large changed-file performance.

## Acceptance criteria

- [ ] Visuals communicate measurable state with negligible refresh overhead.

## Progress evidence (2026-09-22)

- Added pure `internal/ui/activityviz` projections for conflict-weighted change heat, fixed-width ASCII diff bars, and bounded recent-value sparklines. They clamp inputs, cap output width, and retain numeric/text semantics for no-color and accessibility contexts.
- Integrated the projections into repository dashboard rows using existing staged, unstaged, untracked, conflict, ahead, and behind metrics; no additional Git/provider/filesystem work is performed during rendering.
- Added bounded daily commit buckets from already-loaded `HistoryCommits` for the active repository dashboard row; history loading remains the existing bounded path and rendering only consumes the resulting counters.
- Added unit coverage for heat weighting, bar clamping, and bounded sparkline behavior.
- Remaining: optional/configurable dashboard space, large changed-file performance evidence, and native 80x24/NO_COLOR/reduced-motion acceptance.

- Shared status presentation coverage now includes an explicit 80x24,
  `NO_COLOR`, and full/reduced/off motion regression over 14,953 logical
  entries. Visualization-specific native acceptance and configurable dashboard
  space remain open.

- Revision `c25d6de` wires the validated `visuals.enabled` and
  `visuals.activity_buckets` settings into the repository dashboard. Disabled
  visuals now reclaim dashboard space, while bucket widths are clamped to the
  schema's safe 1–32 range. Focused view/app tests and the full `make check`
  gate pass; large changed-file evidence and native 80x24/NO_COLOR/reduced-
  motion acceptance remain open.

- Revision `d9f2fe2` adds Darwin arm64 PTY evidence for the real 80x24
  dashboard/status lane and 14,953-file status scenario. The run used
  reduced motion and the repository's bounded PTY harness; visualization-
  specific NO_COLOR and Linux/Windows acceptance remain open.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
