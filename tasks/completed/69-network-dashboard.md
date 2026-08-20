# Task 69: Create remote synchronization dashboard

Status: Complete

Progress: Added a render-neutral dashboard model with stale-fetch detection, active job filtering, branch divergence fields, and bounded recent activity. The workspace now provides an asynchronous `n` remotes route with selectable remote rows, state rendering, fetch, explicit-strategy pull, push preview/confirmation, and force-with-lease confirmation; completed remote operations now persist bounded success/failure activity across refreshes. Remote operations now register running jobs, expose active-job state, and support cancellation requests.

## Objective
Create an htop-like remote panel showing branch tracking, ahead/behind, stale fetch indicators, in-flight jobs, last success/failure, and concise network activity. Provide fetch/pull/push actions without leaving the dashboard.

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

- The remotes workspace is a render-neutral dashboard with stale-fetch detection, branch divergence, bounded activity, and active job details.
- Fetch, explicit-strategy pull, normal push, set-upstream push, tag push, and force-with-lease push remain on asynchronous typed command paths; jobs carry cancellation and terminal state.
- Mouse selection is supported alongside keyboard navigation, and failures produce sanitized notifications plus actionable conflict status.
- Tests cover dashboard projection/rendering, stale and active state, mouse selection, cancellation, push confirmations, real bare-remote behavior, race safety, vetting, and formatting.
