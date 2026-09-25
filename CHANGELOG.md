# Changelog

All notable user-visible changes to gitwatch are documented here. The project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

No post-v1.0.9 entries have been recorded here yet.

## [1.0.9] - 2026-09-22

### Changed

- Updated `github.com/mattn/go-runewidth` from `0.0.29` to `0.0.30`, as recorded in the [published release notes](https://github.com/sphireinc/gitwatch/releases/tag/v1.0.9).

## Project feature summary (not version-attributed)

The bullets below are a cumulative project overview inherited from the
pre-launch changelog. They are not an Unreleased list and have not been mapped
to individual tags. See [GitHub Releases](https://github.com/sphireinc/gitwatch/releases)
for per-release notes; future changes should be recorded under `[Unreleased]`
and moved into a dated version section when released.

### Added

- Live authoritative Git status, staged/unstaged diff inspection, conflict-aware file details, safe stage/unstage, watcher/poll fallback, and responsive keyboard/mouse TUI behavior.
- Guarded commit, hunk, stash, branch, worktree, history, fetch, pull, and push workflows.
- Multi-repository dashboards, optional read-only GitHub provider views, notifications, a command palette, and an out-of-process plugin SDK.
- Versioned configuration, terminal capability handling, reduced-motion support, security diagnostics, integration tests, performance budgets, and cross-platform release tooling.
- Bounded path-history and blame inspection, including origin-commit navigation.
- Guarded historical patch editing through an inverse-patch preview and controlled rebase.
- Flat and collapsible tree status presentation with directory aggregation.
- Typed-argv editor, opener, and difftool handoffs with bounded temporary materialization.
- Shell-free custom commands with context restrictions, placeholder validation, bounded output, cancellation, and refresh policy.
- Repository-scoped `.gitignore` management with an offline catalog, previewed byte-preserving edits, managed-block ownership, and concurrent-edit protection.
- Multi-repository health details now separate authoritative local state from fresh/stale provider data and show measured remote-fetch outcome and latency.

### Changed

- Canonical module and installation path is `github.com/sphireinc/git-watch`.
- Public documentation, contributor policy, CI, release packaging, and repository hygiene were prepared for the first FOSS release.
- The production TUI now drives status through the coalescing refresh coordinator and observes both worktree files and linked-worktree Git metadata, with visible polling fallback and clean cancellation during repository switches and shutdown.
- Filesystem watching ignores read-only Git-metadata events that macOS kqueue reports as mode changes, preventing status-refresh feedback loops while preserving real worktree and metadata changes.
