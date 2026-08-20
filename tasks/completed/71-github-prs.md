# Task 71: Add GitHub pull request integration

Status: Complete

Progress: Added provider-neutral pull-request metadata parsing and a TTL cache keyed by repository/branch. API transport feeds the opt-in GitHub workspace, which renders PR metadata, check aggregates, review state, and provides safe open-in-browser and copy-URL actions.

## Objective
For the current repository/branch, show associated PR, state, draft status, checks summary, review state, mergeability when available, base/head, URL, and comments/count metadata. Add open-in-browser and copy-URL actions. Cache/rate-limit API reads.

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

- The `G` workspace asynchronously detects the GitHub remote, loads the current-branch PR through a bounded TTL cache, enriches it with check-run and review summaries, and renders sanitized metadata.
- `o` opens only validated HTTP(S) PR URLs through an OS-specific direct-argv browser command; `y` copies the URL without shell interpolation.
- Provider failures degrade to a visible error state and never expose response bodies or credentials.
- Tests cover parsing/cache behavior, review/check transport, API failure redaction, safe browser command construction, and GitHub workspace routing/rendering.
