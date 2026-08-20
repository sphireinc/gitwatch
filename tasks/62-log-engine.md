# Task 62: Build scalable commit history engine

Status: In progress

Progress: Added bounded NUL/record-delimited log parsing with SHA, parents, author, timestamp, subject, refs, and signature fields, plus skip/limit pagination with a one-record lookahead. The TUI now loads history pages asynchronously and appends additional pages from the History workspace while preserving the graph selection; leaving History cancels an in-flight page request so stale results do not update the workspace.

## Objective
Load commit history incrementally using stable machine-readable formatting. Model SHA, parents, refs, author, timestamp, subject, signature state, and graph topology. Support pagination/lazy loading and cancellation for very large histories.

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
