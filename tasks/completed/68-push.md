# Task 68: Implement push workflows

Status: Complete

Progress: Added explicit branch push and opt-in force-with-lease command primitives with validated remote/ref arguments, plus a typed `PreviewPush` operation that resolves local and remote SHAs through Git's structured commands. The remotes workspace asynchronously previews and confirms normal pushes, supports `u` set-upstream pushes, and requires explicit `P` then `y` confirmation before force-with-lease; `T` enters an explicit tag name and confirms tag push. A real bare-remote integration scenario covers new-ref preview, push, post-push preview, and fetch.

## Objective
Add push current branch, set-upstream push, tag push where explicitly selected, and force-with-lease behind strong confirmation. Never offer raw --force as the default. Preview local/remote ref movement before destructive/non-fast-forward operations.

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

- Normal pushes retain typed local/remote SHA preview and confirmation; force-with-lease is separately gated behind `P` then `y` and raw `--force` is never offered.
- `u` performs an explicit `--set-upstream` push for the selected remote and current branch.
- `T` accepts one explicit validated tag name and pushes it using an explicit tag refspec after confirmation; tag and remote names reject option-like/control-containing values.
- Successful pushes continue through the authoritative status and remote metadata refresh path, while failures preserve actionable notifications/activity.
- Tests cover validation, push previews, real bare-remote push/fetch behavior, set-upstream/tag controls, and repository-wide race/vet/build gates.
