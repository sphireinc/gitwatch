# Task 70: Add optional GitHub authentication/provider layer

Status: Complete

Progress: Added GitHub remote detection, optional environment-backed token loading, direct-argv GitHub CLI credential reuse with environment fallback, and an optional context-aware GitHub API client for pull requests and check runs; token values are never persisted or formatted, and provider errors omit response bodies. The opt-in GitHub workspace now loads and renders provider data asynchronously with graceful offline/error handling.

## Objective
Detect GitHub remotes and implement an optional provider abstraction. Prefer GitHub CLI credential reuse when available; otherwise support secure token configuration without writing secrets to normal config/logs. Core gitwatch must remain fully functional when GitHub integration is disabled.

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

- GitHub support is disabled unless `github.enabled` is true; core status and Git workflows remain independent of provider availability.
- The `G` workspace detects the first GitHub remote, prefers `gh auth token`, then falls back to the configured token environment variable, and never writes credentials to config, logs, or rendered output.
- PR/check reads run asynchronously through the typed provider client and are rendered through a sanitized GitHub view. PR actions and browser integration remain scoped to Tasks 71 and 72.
- Tests cover remote parsing, token-source fallback, transport/error redaction, provider parsing/cache behavior, async app routing, and sanitized UI rendering.
