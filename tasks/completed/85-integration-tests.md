# Task 85: Build end-to-end post-v1 Git scenario suite

Status: Complete

Progress: Added real temporary-repository scenarios covering initialization, commit, snapshot refresh, stash creation/listing, branch creation, worktree creation/discovery/removal, occupancy mapping, bare-remote push-preview/push/fetch, partial staging/discard, pre-commit hook failure/output, divergent merge conflict snapshots, and multi-repository refresh transitions through the registry engine. Git argument-vector execution and hostile terminal rendering remain covered by architecture/security tests; the real-repository scenarios use portable Git commands and skip no required behavior.

## Objective
Create temporary real repositories covering commits, hooks, partial staging, stashes, conflicts, branches, remotes, worktrees, merge graphs, fetch/pull/push, and multi-repo state. Assert both Git outcomes and application model transitions.

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
- Implementation: end-to-end temporary repositories exercise Git outcomes and application snapshots across the v1 workbench workflows.
- Multi-repository coverage: `TestMultiRepositoryRefreshTransitionScenario` verifies clean and changed repositories are refreshed concurrently and mapped back to the correct registry rows.
- Tests: integration package plus repository-wide test, race, vet, formatting, and security gates pass on the development platform.
- Deferred follow-up: maintainers should repeat the real-repository suite on each supported OS in release CI; no platform-specific exception is required by the test code.
