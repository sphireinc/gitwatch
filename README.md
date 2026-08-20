# gitwatch

![gitwatch logo](assets/gitwatch-logo.png)

An htop-style interactive Git worktree dashboard for the terminal. gitwatch continuously reflects authoritative Git status, shows staged/unstaged state and branch health, and lets you inspect diffs and safely stage or unstage paths without leaving the terminal.

## Install

Requires Go 1.25+ for source installation and Git on `PATH`.

```sh
go install github.com/jusanchez/gitwatch/cmd/gitwatch@latest
gitwatch --help
```

Release archives and SHA256SUMS are produced by `make release VERSION=0.1.0`. The binary reports its build identity with `gitwatch --version`.

## Quick start

Run `gitwatch` inside any normal Git worktree, including a nested directory. Select a status row with arrows or `j`/`k`; click a row to open its diff/details pane on the right in wide terminals. Press `Enter` or `d` for the keyboard equivalent, Space to stage/unstage, `b` for branches, `s` for stashes, `l` for history, and `q` to quit.

## Keymap and status symbols

See [KEYMAP.md](KEYMAP.md) for the complete default keymap. `S` means staged, `M` modified, `?` untracked, `!` conflict, `D` deleted, and `R` renamed. Symbols remain meaningful with `NO_COLOR=1`.

Advanced workflow notes are in [docs/advanced-workflows.md](docs/advanced-workflows.md), including history, patch, remote, GitHub, plugin, and multi-repository safety semantics.

## Configuration and watch modes

Configuration is JSON at `$XDG_CONFIG_HOME/gitwatch/config.json` or the platform fallback. `GITWATCH_CONFIG`, `GITWATCH_THEME`, `GITWATCH_MOTION`, `GITWATCH_WATCH`, and `GITWATCH_INTERVAL` provide explicit environment overrides; CLI flags take precedence. Watch modes are `auto`, `fs`, and `poll`. Motion is `full`, `reduced`, or `off`.

## Safety

Git is invoked through argv arrays, never shell strings. Paths are passed after `--` where supported. Staging and unstaging are reversible and refresh from Git afterward. Restore requires a confirmation naming the exact path and affected content; generic `reset --hard` and `clean -fd` actions are not available.

## Troubleshooting and architecture

Gitwatch requires an external Git executable and rejects bare repositories in v1. If filesystem notifications fail, `--watch=poll` or the configured poll mode keeps status current. See [ARCHITECTURE.md](ARCHITECTURE.md), [docs/performance.md](docs/performance.md), and [docs/edge-cases.md](docs/edge-cases.md) for design and validation details.

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md). Security reports belong in [SECURITY.md](SECURITY.md).

## Demo recording

Run `./scripts/demo-repo.sh /tmp/gitwatch-demo` and follow [docs/demo.md](docs/demo.md) for a deterministic mixed-status recording fixture.

## Product
`gitwatch` is an htop-style interactive Git worktree dashboard for the terminal. It continuously reflects repository state, makes Git status understandable at a glance, and lets the user perform safe day-to-day Git actions directly from a rich TUI.

The v1 experience must feel alive: responsive keyboard and mouse interaction, tasteful animation, clear state transitions, immediate feedback, dense information without visual clutter, and safe Git operations.

## Locked stack
- Language: Go 1.25+
- TUI: `charm.land/bubbletea/v2`
- Components: `charm.land/bubbles/v2`
- Styling/layout: `charm.land/lipgloss/v2`
- Filesystem notifications: `github.com/fsnotify/fsnotify`
- Git integration: invoke the system `git` executable; do not implement Git object/index semantics in Go for v1.
- Parsing contract: prefer stable machine-readable Git output such as `git status --porcelain=v2 -z --branch`.
- Distribution: standalone static-ish binaries where platform permits; Git itself remains an external runtime requirement.
- Platforms at v1: macOS, Linux, Windows.
- License: MIT unless repository owner explicitly selects another license before Task 01.

## Non-negotiable UX
The default screen should resemble a polished process monitor rather than a simple list wrapper. It must expose branch/upstream state, divergence, counts, file states, staged/unstaged split, repository health/activity, selection details, context-sensitive actions, operation feedback, and a compact event timeline.

Users must be able to select a file and stage/unstage it without leaving the app. Mouse support is required, but every action must have a keyboard equivalent.

## Safety model
Never execute destructive Git actions silently. Staging/unstaging are immediate and reversible. Discard/reset/clean/restore/delete operations require an explicit confirmation UI that identifies exactly what will change. Shell interpolation of filenames is forbidden; commands must use argument arrays and `--` where applicable.

## Definition of v1
v1 is launchable only when all P0 tasks are complete, acceptance criteria pass on macOS/Linux/Windows, release artifacts are reproducible, README docs are complete, and the application has no known data-loss bug.

## Task order
Tasks are numbered in intended implementation order. A later task may be started early only when it does not change interfaces owned by an unfinished prerequisite. Record dependency exceptions in `DECISIONS.md`.
