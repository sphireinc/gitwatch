# Task 76: Implement multi-repository discovery and registry

Status: Complete

Progress: Added bounded filesystem discovery for explicit roots, nested normal/worktree repositories, ignored and symlinked directories, depth/repository limits, and a private JSON registry with favorite/group/last-opened metadata. The `v` repository dashboard now loads and merges registry metadata asynchronously, applies configured groups, persists discovered records, and updates last-opened state when opening a repository with authoritative discovery/refresh.

## Objective
Support explicitly configured roots plus bounded directory scanning for repositories/worktrees. Persist a lightweight registry with path, display name, last-opened time, favorite state, and optional groups. Respect ignore rules and avoid scanning huge trees without limits.

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

- Discovery is bounded by configured roots, depth, repository count, ignore rules, cancellation, and symlink safety.
- Repository metadata is persisted privately and merged into asynchronous dashboard discovery without replacing stored favorite, group, name, or last-opened fields.
- Configured groups are applied during registry merge; opening a dashboard row records `last_opened` and preserves the registry for subsequent launches.
- Registry merge, private-file, discovery-limit, symlink, and app/dashboard flows are covered by tests; repository-wide test, race, vet, and diff gates pass.
