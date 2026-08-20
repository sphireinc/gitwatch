# Task 82: Extend configuration and keybinding system

Status: Complete

Progress: Added versioned v2 module configuration for repositories, remotes, GitHub, plugins, and keymaps; defaults, limit/duration validation, binding collision detection, and `--config-check` are implemented. Loading now migrates unversioned/v1 files in memory and rejects future versions; the complete schema and migration behavior are documented. Configured quit/help/navigation bindings now dispatch through the runtime app keymap.

## Objective
Add structured config for new modules, remote behavior, GitHub, plugins, repo roots/groups, refresh budgets, animation, and keymaps. Provide schema validation, migrations, `gitwatch config check`, and collision detection for bindings.

## Required implementation
- Produce production-quality implementation, not a prototype.
- Integrate with the existing Bubble Tea message/update architecture and typed Git runner.
- Keep the UI responsive; blocking filesystem, Git, network, and provider work must not run in the render/update hot path.
- Add keyboard and mouse behavior where the task introduces an interactive surface.
- Add structured errors/activity events and refresh affected repository state after mutations.
- Add focused unit/integration tests for success, failure, cancellation, and relevant edge cases.
- Update help/keymap/config/docs when this task adds user-visible behavior.

## Acceptance criteria
- Feature works on macOS, Linux, and Windows unless the task explicitly documents a platform limitation.
- No shell-string interpolation is introduced for Git/process execution.
- User-controlled terminal text is sanitized against control/escape injection.
- Existing v1 status/stage/diff workflows remain functional.
- `go test ./...`, static analysis, and formatting checks pass.
- The task is not complete until automated tests cover its primary behavior.

## Completion artifact
Implementation notes:

- Configuration remains versioned at v2 with in-memory migration for unversioned/v1 files and rejection of future versions.
- Repository, remote, GitHub, plugin, motion, and keymap settings are validated before use; duplicate bindings are rejected.
- Runtime keymap bindings merge with canonical defaults and dispatch through the existing Bubble Tea action paths, including navigation, refresh, help, and quit.
- `--config-check`, configuration documentation, migration tests, collision tests, and remapped runtime dispatch tests are included.
