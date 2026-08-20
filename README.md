# gitwatch — v1 Build Task Pack

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
