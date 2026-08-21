# Task 88: Write advanced user documentation

Status: Complete

Progress: Added advanced workflow documentation covering history/patches, stash/branch/worktree safety, remotes/GitHub, plugins, multi-repository behavior, v2 configuration, security commands, a v1 migration guide, operator troubleshooting, and updated branch/notification keymaps. Added a deterministic demo/recording guide that includes the file-to-diff-pane workflow and narrow-terminal behavior.

## Objective
Document commit composer, patch staging, stash/branch/worktree/history/remotes, GitHub, plugins, multi-repo dashboards, safety semantics, keymaps, configuration, troubleshooting, and migration from v1. Include terminal recordings/screenshots where useful.

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
Record implementation notes, key decisions, new commands/keybindings/configuration, tests added, and any deliberately deferred follow-ups in the task/PR completion summary.

## Completion summary
- Documentation: advanced workflows, configuration, edge cases, performance, plugins, security, migration, troubleshooting, release checks, and demo/recording guidance are published under `docs/`.
- User-visible references: `README.md` and `KEYMAP.md` link the advanced guides and document branch, notification, and file-diff interactions.
- Verification: documentation changes pass repository-wide tests, race, vet, formatting, and release/security scripts.
- Deferred release activity: recordings and platform-specific operator evidence are called out in the demo and release checklists for maintainers to capture on supported terminals.
