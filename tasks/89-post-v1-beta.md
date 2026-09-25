# Task 89: Run feature-complete beta hardening

Status: In progress

Progress: Repository-wide tests, race tests, vetting, benchmarks, release checks, strict five-target artifact verification, OS-specific CI runtime smoke checks, and real-repository integration fixtures pass; the plugin API-1 compatibility fixtures, beta validation matrix, feedback template, and release sign-off record are documented. The matrix is owner-signed for exact candidate `5b2a8e9` across macOS, Windows, and all Linux cells by explicit platform-equivalence acceptance; physical multi-distribution Linux testing continues. On 2026-09-25 the owner confirmed no known private/unreported blocker, critical, or data-loss issue beyond the public tracker, which showed zero open issues. The owner selected beta tag `v1.1.0-beta.1`; candidate disposition, publication, and the feedback window remain open, and no beta tag or release has been created.

## Objective
Cut a beta release containing all post-v1 features. Collect crash/error/performance feedback, test across macOS/Linux/Windows and major terminals, validate Git versions, freeze plugin API candidate, and resolve all release-blocking defects.

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

## Beta closure and publication status (2026-09-25)

The owner-provided matrix disposition covers all listed terminal, OS, Git,
workbench, integration, and workload cells for candidate `5b2a8e9`; Linux is
accepted by explicit macOS equivalence, not represented as physical Linux
testing. The owner also confirmed that no privately tracked or unreported
blocker, critical, or data-loss issue is known beyond the public tracker.

The local tag list currently contains `v1.0.9` through `v1.0.0`, and GitHub's
release list likewise shows `v1.0.9` as the latest publication. That stable
release targets `951f3f6`, 200 commits and 89 Go source files before candidate
`5b2a8e9`; it is not a post-v1 feature beta. Current tested source `34562b5`
differs from `5b2a8e9` only in merge-engine error propagation and its test.
The user-approved carry-forward to `34562b5` is recorded for Tasks 137 and
143, but not yet for this beta-wide task. Do not claim the later disposition
or the v1.0.9 stable publication satisfies Task 89's exact-candidate beta
release requirement.

**Still required:** resolve whether the all-cell matrix disposition carries to
the exact release commit, then create and publish the authorized
`v1.1.0-beta.1` tag/release and complete its feedback window. No beta tag or
release has been created.

## Prerelease channel safeguard (2026-09-25)

The release workflow now classifies `vMAJOR.MINOR.PATCH` tags as stable and
hyphen-suffixed versions such as `v1.1.0-beta.1` as prereleases. It passes
GitHub CLI's `--prerelease` flag both when creating a release and when updating
an existing one, so a beta tag is not accidentally published in the stable
channel. `scripts/release-policy-check.sh` covers channel classification,
invalid tags, and workflow wiring; it is part of `make check`, CI, and the
release check. `docs/distribution.md` documents the distinction.

Local verification on Darwin arm64 / Go 1.27.0: release policy checks, release
workflow YAML parsing, extracted publish-step Bash syntax, formatting,
`go test ./...`, `go test -race ./...`, `go vet ./...`, security checks, and
performance budgets passed. The repository-pinned golangci-lint v2.12.0 was
built from its cached source, but local lint could not type-check Go 1.27's
new generic standard-library methods (`math/rand/v2`) with the Go 1.25.5-built
linter; the required Go 1.25.10 toolchain is not installed in this restricted
environment. Hosted CI run [36178722786](https://github.com/sphireinc/gitwatch/actions/runs/36178722786)
passed the pinned lint, release-channel policy check, full-history secret scan,
and Linux, macOS, and Windows jobs. On clean commit `a121687`,
`VERSION=1.1.0-beta.1 ./scripts/release-check.sh` also passed on Darwin arm64 /
Go 1.27.0, including isolated source install/runtime smoke, full tests, race,
vet, security/performance checks, five-target archives, release metadata, and
checksum verification. This does not publish a beta. The owner selected
`v1.1.0-beta.1`, but whether the recorded matrix disposition carries to the
exact release commit still needs resolution, so no beta tag or release has
been created.

## Completion artifact
Record implementation notes, key decisions, new commands/keybindings/configuration, tests added, and any deliberately deferred follow-ups in the task/PR completion summary.
