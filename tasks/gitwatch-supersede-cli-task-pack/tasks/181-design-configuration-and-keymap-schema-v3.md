# Task 181: Design configuration and keymap schema v3

**Phase:** Hardening
**Depends on:** 121, 167, 177

**Status:** Complete for the configuration task's acceptance scope. Typed v3
configuration, deterministic read-only migration, schema and migration
goldens, invalid-field rejection, safe keybindings, and inspection redaction
are implemented and tested. This task changes no TUI interaction; wider
publication and native release acceptance remain in the release tasks.

## Goal

Version configuration deliberately for advanced workbench features instead of accumulating ad-hoc keys.

## Non-negotiable constraints

- Live filesystem-driven status is the product core. Do not replace, subordinate, or pause it except for the minimum repository-lock window required by Git itself.
- Filesystem events are refresh hints, never authoritative state. The authoritative worktree snapshot remains `git status --porcelain=v2 -z --branch --untracked-files=all` parsed into immutable repository state.
- Every successful mutation MUST request an authoritative refresh for the affected repository. Long-running sequencer operations must refresh after every observable state transition.
- Multi-repository support is first-class. New domain models and operations MUST carry repository identity/scope and remain correct while other repositories refresh or run unrelated work.
- Do not create unbounded watchers, goroutines, workers, or Git/provider/plugin processes. Reuse bounded registry/operation infrastructure.
- All Git commands use typed argv execution through the Git boundary. Never interpolate repository data into shell command strings. Use `--` where supported and machine-readable/NUL-delimited output where available.
- Bubble Tea owns UI state. Git/network/filesystem/process work never runs in the render path.
- Repository-controlled text is untrusted terminal input and MUST be sanitized before rendering.
- Destructive/history-rewriting actions require scope-specific confirmation. Keep the prohibition on generic `reset --hard`, raw `--force`, and `clean -fd` shortcuts.
- Keyboard and mouse must reach equivalent functionality. New views must work at 80x24, honor `NO_COLOR`, and support full/reduced/off motion.
- Do not reimplement Git. Use Git as source of truth and build safe typed control/presentation layers around it.
- Breaking config/plugin changes require versioning, migration, and compatibility fixtures.

## Implementation steps

1. Define schema v3 for new workspaces, custom commands/forms, external tools, auto-fetch, provider options, history/reflog budgets, multi-repo limits and keybindings.
2. Provide deterministic v2→v3 migration that preserves existing watcher mode/interval, themes, motion, plugins, registry groups/favorites and keymaps.
3. No migration may silently disable filesystem watching or make polling the default.
4. Validate duplicate/unsafe bindings and reserved terminal controls.
5. Update `--config-check` and redacted `--config-inspect`.
6. Freeze migration fixtures and document compatibility boundary.

## Verification

- Golden migration fixtures, malformed fields, secrets absent from inspect.

## Acceptance criteria

- [x] Existing users upgrade without losing watcher/multi-repo/plugin behavior.
- [x] New advanced config is typed and validated.

## Progress evidence (2026-09-22)

- Advanced configuration has been promoted to schema version 3 with typed `workspace` bounds (`palette_max_results`, `status_overscan`) and typed `visuals` activity-bucket settings.
- Deterministic v2-to-v3 in-memory migration reporting preserves watcher mode, interval, plugin enablement, repository limits, keymaps, and all existing v2 fields; source files remain read-only.
- v3 defaults and bounds are enforced by `Validate`, exposed in the machine-readable JSON schema, and consumed by palette/status presentation limits.
- Added v2 migration fixture coverage plus malformed/limit validation through the existing configuration tests.
- Remaining: full config-inspect redaction audit, golden schema fixture review, and release/native acceptance evidence.

- `GOCACHE=/tmp/gitwatch-config-cache go run ./cmd/gitwatch --config-inspect`
  emitted schema version `3` with typed workspace, visuals, watcher, remote,
  plugin, and keymap settings. The inspection output exposed only the GitHub
  token environment-variable name (`GITHUB_TOKEN`) and no token/password
  value. Golden schema review and native/release acceptance remain open.

- `config.Inspect` now recursively redacts token/password/secret/credential
  fields and inline credential markers in argv arrays while retaining the
  configured token environment-variable name. Regression coverage verifies a
  literal custom-command token cannot appear in inspection output. Golden
  schema review and native/release acceptance remain open.

- Added `TestDocumentedSchemaV3CoversAdvancedConfigurationSurface`, which
  parses the checked-in schema and guards the version identity, workspace and
  visual bounds surface, auto-fetch policies, plugin/tool/custom-command
  sections, keymap sections, and the absence of secret-value properties. The
  test passes in normal, race, and vet-focused runs; native/release acceptance
  remains open.

- The schema test now freezes the checked-in v3 document with a SHA-256 golden
  identity (`90613e7f4df1778e5a5aaf18f1ea1327cd8d801af94aeb45af2248d8ec915808`)
  in addition to its semantic surface checks. At revision `c8fbcf3`,
  focused config tests pass normally and under race/vet; native/release
  acceptance remains open.

## Progress evidence (2026-09-25)

- The configuration guide now enumerates schema-v3 defaults, nested fields,
  environment variables, CLI overrides, duration units, and migration behavior.
- Added `custom_commands[].prompts[]` to the published v3 JSON Schema, including
  required identity fields, prompt kinds, option sources, validation patterns,
  and defaults. The semantic schema test now verifies all prompt fields and
  supported kinds; focused `internal/config` tests pass.
- Updated the schema SHA-256 golden to
  `90613e7f4df1778e5a5aaf18f1ea1327cd8d801af94aeb45af2248d8ec915808`.
  Focused `internal/config` tests and `make check` pass on Go
  `go1.27.0 darwin/arm64`; release-pinned Go and native/release acceptance
  remain open.
- Hosted run `36096357394` for `be0aca1` passed Ubuntu 24.04, macOS 15,
  Windows 2025, quality/policy, and full-history secret scanning. This records
  hosted Go 1.25.10 matrix coverage; publication and native release gates
  remain open.

- Revision `852eb3e` closes the automated inspection-redaction gap for split
  credential argv values such as `--token secret`, bearer/basic authorization
  forms, and colon-delimited credential markers. Focused config tests and the
  full `make check` gate pass, including lint, race, vet, security fuzz checks,
  and performance benchmarks. Native/release acceptance remains open.

## Progress evidence (2026-09-28)

- Extended `--config-inspect` redaction to scalar configuration strings as well
  as argv arrays, covering URL user-info, bearer/inline credential markers, and
  split API-key/client-secret flags. Regression tests assert credential
  absence while retaining ordinary command configuration.
- Refreshed the v3 schema SHA-256 golden to
  `485684f7923c83fac85631d69aadfcb27ee8be768b868dfd5e6ed319be375373` and
  added schema assertions for non-negative integer prompt length bounds and
  zero-only bounds on confirmation and selection prompts.
- Added a historical v2 input fixture and a paired v3 `--config-inspect`
  golden generated from the current loader. The v2 source contains no v3
  workspace/visual fields or later prompt length fields; tests verify the
  source remains unchanged and existing v2 settings survive normalization.
- On pre-commit worktree based at `fbd0728`,
  `GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache go test ./internal/config`
  and `go vet ./...` pass; `gofmt` and `git diff --check` pass. The full
  `make test` attempt failed in `internal/app` because `httptest` could not
  bind its loopback listener (`operation not permitted`). The full race run
  was interrupted to let the main full-suite run proceed. These are local
  worktree results, not commit-level validation; the main agent should record
  its final suite result against the final source before committing.
- The orchestrator's final assembled-source run passed
  `GOCACHE=/tmp/git-watch-go-cache GOMODCACHE=/tmp/git-watch-go-mod-cache make check`
  on Darwin arm64 / Go 1.27.1 after all agent edits were finished. It includes
  pinned golangci-lint v2.12.0 (0 issues), full tests, full race tests, vet,
  formatting, security fuzz checks, performance budgets, and release policy.
  The sandbox listener/download failures above are superseded by this
  successful run with the required test permissions. These are pre-commit
  worktree changes based on `fbd0728`; hosted CI is recorded separately.

## Completion record

- [x] Implementation commit recorded (`14ed506`, redaction follow-up `852eb3e`, current prompt-schema follow-up `9b1cb34`).
- [x] Exact tested revision recorded (`9b1cb342fffba50675ce83dfc957ac0cabae1ea9`, Darwin arm64).
- [x] Focused unit/integration tests recorded.
- [x] `go test ./...` recorded.
- [x] Race/vet/lint/format evidence recorded where applicable.
- [x] Native/manual evidence recorded where this task changes terminal interaction (not applicable; no TUI interaction changed).
- [x] Known limitations/deferred work documented.

- Current-main schema follow-up: the prompt defaults and option constraints
  changed the documented v3 schema digest to
  `cded805c00c7783ae2b04b6d22ffea69c938cc936e98340ef99f7daa3fede2ae`.
  Full `make check` passed at `9b1cb342fffba50675ce83dfc957ac0cabae1ea9`
  on Darwin arm64 / Go 1.27.0. Release-wide platform acceptance remains
  tracked separately by the beta/release gates.
