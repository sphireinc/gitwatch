# Task 83: Threat-model post-v1 surfaces

Status: Complete

Progress: Added centralized secret redaction for logs/diagnostics and final TUI rendering, URL userinfo redaction, hostile terminal/file/diff fixtures, an explicit discovery policy that skips symlinked filesystem entries and rejects symlinked `.git` metadata, bounded plugin protocol decoding, hostile message-size fixtures, an actual hostile out-of-process plugin output fixture, and a security threat-model document. Automated security evidence is enforced by the release check.

Automated security evidence: `scripts/security-check.sh` rejects shell-string/interpolated process execution patterns and runs the hostile platform/plugin/registry test packages; it is now part of `scripts/release-check.sh`. Human release/operator review remains explicitly unverified.

## Objective
Threat-model patches, commit messages, refs, remotes, credentials, GitHub responses, plugin RPC, repository discovery, terminal escape injection, symlinks, and malicious repository contents. Add redaction and hostile-fixture tests.

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
- Threat model: repository contents, terminal escapes, secrets, Git/provider responses, symlinks, process execution, and plugin protocol boundaries are documented in `docs/security.md`.
- Enforcement: `scripts/security-check.sh` rejects shell-string process execution patterns and runs hostile platform/plugin/registry tests; `scripts/release-check.sh` invokes it.
- Tests: repository-wide test, race, vet, formatting, and security gates pass.
- Deferred release activity: the operator must still review the threat model on each supported platform before publication; this is recorded in `docs/release-checklist.md` and is not an unautomatable implementation criterion.
