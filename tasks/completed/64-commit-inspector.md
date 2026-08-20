# Task 64: Build commit inspector and historical diff views

Status: Complete

Progress: Added parent-relative commit inspection primitives returning changed paths, add/delete/binary stats, full patches, and path-safe filtering, plus commit metadata summary. The History workspace now asynchronously inspects the selected commit and renders metadata, file stats, and a bounded patch preview; `M` cycles merge parents, `f` applies a path filter, `g` resolves and jumps to a typed ref, and `y` requests native OSC52 SHA copy.

Completion summary: The inspector renders commit summary, parents, refs, parent-relative merge diffs, changed-file statistics, and bounded patches. `M` cycles merge parents, `f` applies a path filter, `g` resolves a ref and selects it in loaded history, and `y` requests OSC52 SHA copy. Inspection and ref resolution use asynchronous typed Git commands with option-like input rejection. Unit, application-routing, integration, race, and static-analysis gates pass.

## Objective
Selecting a commit should show metadata, parents, refs, changed-file stats, full patch, and parent-relative diff. Add copy-SHA, inspect parent, jump-to-ref, and path filtering. Support merge commits explicitly.

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
