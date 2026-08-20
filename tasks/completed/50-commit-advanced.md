# Task 50: Add amend, signoff, signing and author options

Status: Complete

Progress: Commit execution supports amend, no-edit, signoff, signing, and author options; repository signing/user configuration inspection, multiline-author rejection, and normalized SHA output are implemented. The composer exposes toggles, one-line author editing, amend confirmation, and asynchronously loaded identity/signing guidance.

## Objective
Support --amend, --no-edit where appropriate, Signed-off-by, configured GPG/SSH signing, and explicit author override. Detect repository/user configuration and make dangerous history-rewriting behavior visually distinct with confirmation.

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

- `git.CommitConfig` reads configured user identity, `commit.gpgsign`, and `gpg.format` through the typed runner; the commit workspace loads it asynchronously and sanitizes the displayed summary.
- `A`, `N`, `o`, `S`, and `@` control amend, no-edit, signoff, signing, and author override. Amend requires explicit `y` confirmation and never discards the draft on failure.
- Author input rejects newline/control-line injection, commit SHAs are normalized, and all mutation completion paths refresh repository state.
- Tests cover commit configuration parsing, author validation, composer options, amend confirmation/cancellation, failure preservation, and repository-wide race/vet/build gates.

Deferred follow-up: platform-specific signing-agent prompts remain delegated to Git and are reported through Git's normal command error output.
