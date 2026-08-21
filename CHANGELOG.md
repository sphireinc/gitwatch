# Changelog

All notable user-visible changes to gitwatch are documented here. The project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Live authoritative Git status, staged/unstaged diff inspection, conflict-aware file details, safe stage/unstage, watcher/poll fallback, and responsive keyboard/mouse TUI behavior.
- Guarded commit, hunk, stash, branch, worktree, history, fetch, pull, and push workflows.
- Multi-repository dashboards, optional read-only GitHub provider views, notifications, a command palette, and an out-of-process plugin SDK.
- Versioned configuration, terminal capability handling, reduced-motion support, security diagnostics, integration tests, performance budgets, and cross-platform release tooling.

### Changed

- Canonical module and installation path is `github.com/sphireinc/git-watch`.
- Public documentation, contributor policy, CI, release packaging, and repository hygiene were prepared for the first FOSS release.
- The production TUI now drives status through the coalescing refresh coordinator and observes both worktree files and linked-worktree Git metadata, with visible polling fallback and clean cancellation during repository switches and shutdown.

The first public release will move these entries into a dated `1.0.0` section after the cross-platform operator and publication gates in the release checklist are complete.
