# Task 207: Run the `.gitignore` manager acceptance and release gate

## Objective

Complete a feature-level acceptance pass and refuse release until the workflow is safe, fast, offline-capable, and multi-repo aware.

## Non-negotiable constraints

- **Do not weaken gitwatch's live model.** Filesystem events are hints; the authoritative repository state remains the existing `git status --porcelain=v2 -z --branch` refresh pipeline. Editing `.gitignore` must feed back into that pipeline immediately and must not create a second status model.
- **Multi-repository support is mandatory.** Every state object and mutation must be scoped by repository identity/path. Never assume one global active repository.
- **Preserve user-authored `.gitignore` content byte-for-byte wherever it is not intentionally changed.** Do not sort, normalize, wrap, reformat, or rewrite unrelated content.
- **Never shell-interpolate paths or template names.** Reuse gitwatch's argv-based process runner and existing path/safety boundaries.
- **All writes require previewable intent and race protection.** Re-read/hash the target before write; abort if it changed after preview.
- **Managed template removal must be exact and reversible.** Never delete hand-written rules merely because they resemble an upstream template.
- **The feature must work offline.** The embedded catalog is always usable; runtime upstream refresh is additive and optional.
- **Do not make UI rendering perform filesystem, network, or Git work.** Background commands return Bubble Tea messages into the model.

## Context and required behavior

This task is a hard gate, not a documentation checkbox. The feature is done only when a user can start from a fresh repo, create a composed `.gitignore`, continue live work, later append/remove/update combinations, and perform the same operations safely from multi-repo mode.

## Implementation steps

1. Execute the full automated test suite, race-sensitive tests, lint/vet, and platform CI on Linux, macOS, and Windows.
2. Manual acceptance: fresh repo → open manager → search `php` → select PHP plus at least one other template → preview → create → verify Git status immediately reflects ignored files without manual restart.
3. Manual acceptance: existing handwritten `.gitignore` → detect a full unmanaged match at top with `*` → append a new managed template → remove only the managed template → prove handwritten bytes remain unchanged.
4. Manual acceptance: two managed templates with overlapping rules → remove one → verify the other block and behavior remain intact.
5. Manual acceptance: stale/older managed block → load newer catalog → preview update → apply only block-local diff.
6. Manual acceptance: multi-repo dashboard → select several repos → batch add one template → preview per repo → execute → inspect partial failure report and per-repo live refresh.
7. Manual acceptance: external process edits `.gitignore` after preview but before apply → gitwatch refuses to overwrite and requests re-preview.
8. Manual acceptance offline: disable network/cache → browse/search full bundled catalog and create/remove templates.
9. Update release notes and capability matrix only after all gates pass.

## Expected code areas

- `docs/release/*`
- `test/integration/gitignore/*`

## Acceptance criteria

- [x] All platform CI passes.
- [x] Fresh-repo creation works with multi-select search (owner-accepted by source equivalence; see disposition below).
- [x] Existing handwritten content survives append/remove flows exactly (owner-accepted by source equivalence; see disposition below).
- [x] Overlap removal is reversible/safe (owner-accepted by source equivalence; see disposition below).
- [x] Multi-repo batch behavior is proven (owner-accepted by source equivalence; see disposition below).
- [x] Concurrent edits are protected (owner-accepted by source equivalence; see disposition below).
- [x] Offline use is proven (owner-accepted by source equivalence; see disposition below).
- [x] Live filesystem-driven Git status remains the primary post-mutation truth (owner-accepted by source equivalence; see disposition below).

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.

## Automated gate and release evidence

- `VERSION=1.0.0 ./scripts/release-check.sh` passed on macOS arm64 with Go 1.27.0, including full tests, race tests, vet, performance, security, isolated install/config smoke, demo-repository status, cross-target archive generation, SBOM packaging, and artifact verification.
- `./scripts/secret-scan.sh --history` passed with no leaks found across 369 commits.
- `make check` passed on macOS arm64 with Go 1.27.0 after fixing all 21 golangci-lint findings and making disposable-commit tests independent of host-wide Git signing configuration. This includes formatting, lint (0 issues), full tests, race tests, vet, security, and performance gates.
- General native operator acceptance remains tracked by `docs/release-checklist.md`, including terminal/UI, filesystem notification, resize, installation, and shutdown checks. Those repository-wide release rows are separate from this task's feature-specific acceptance criteria and do not reopen them.

## Current-main automated revalidation (2026-09-25)

On Go 1.27.0 / Darwin arm64, the focused packages passed:
`go test ./internal/gitignore/... ./internal/ui/gitignoreview ./internal/multirepo`.
Hosted Actions run [36178722786](https://github.com/sphireinc/gitwatch/actions/runs/36178722786)
passed pinned lint, the full-history secret scan, full tests/race/vet, and
Ubuntu, macOS, and Windows jobs for commit `a121687`; the subsequent changes
through current `main` `a443231` are docs/release-workflow only, with no Go
source changes. These automated results do not replace Task 207's specific
operator flows (fresh-repo composition, handwritten-byte preservation,
overlap removal, multi-repo batch behavior, stale-preview protection, and
offline operation). The owner-provided beta matrix sign-off is recorded
separately and is not represented as a retained scenario-by-scenario Task 207
run log. Task 207 remains open pending explicit linkage/carry of operator
acceptance for those feature-specific scenarios.

## Latest verification (2026-09-25)

Hosted Actions run [36179909264](https://github.com/sphireinc/gitwatch/actions/runs/36179909264)
for `eebde8f` completed successfully with quality/policy, full-history secret
scanning, and all three OS matrix jobs. The `.gitignore` manager source is
unchanged between `eebde8f` and local `main` `ab4b234`; the non-publishing
`VERSION=1.1.0-beta.1 ./scripts/release-check.sh` also passed on `ab4b234`.
This closes the hosted platform-CI criterion for the unchanged feature code,
not the native feature-specific scenarios. Fresh-repo creation, exact
handwritten-byte preservation, overlapping-template removal, multi-repo batch
behavior, stale-preview protection, and offline operation still require
auditable operator evidence or explicit owner closure.

## Beta candidate CI revalidation (2026-09-26)

Hosted Actions run [36265558585](https://github.com/sphireinc/gitwatch/actions/runs/36265558585)
completed successfully for exact commit `9b1cb342fffba50675ce83dfc957ac0cabae1ea9`.
All five jobs passed: quality/policy, full-history secret scan, and test jobs
for Ubuntu 24.04, macOS 15, and Windows 2025. The `.gitignore` manager source
is unchanged at this candidate. At this point the seven feature-specific
operator criteria still awaited owner disposition; the result did not itself
provide native/operator transcripts.

## Owner acceptance by source equivalence (2026-09-27)

**Status: Complete.** The owner explicitly accepted carrying the all-platform
disposition to this task because the `.gitignore` manager application source is
unchanged from `5b2a8e9` through exact candidate `9b1cb34`. This closes each of
the seven feature-specific criteria above by owner acceptance: fresh-repo
composition, handwritten-byte preservation, overlap removal, multi-repo batch
behavior, concurrent-edit protection, offline use, and authoritative live
status refresh. It is not a claim that fresh native runs or
scenario-by-scenario transcripts were collected; those transcripts remain
absent. Exact-candidate hosted CI passed in run `36265558585`, and the
published beta release workflow passed in [run 36307343578](https://github.com/sphireinc/gitwatch/actions/runs/36307343578).
The repository-wide native operator rows in `docs/release-checklist.md` remain
a separate open release gate.
