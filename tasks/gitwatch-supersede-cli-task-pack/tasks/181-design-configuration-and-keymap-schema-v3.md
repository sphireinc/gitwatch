# Task 181: Design configuration and keymap schema v3

**Phase:** Hardening
**Depends on:** 121, 167, 177

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

- [ ] Existing users upgrade without losing watcher/multi-repo/plugin behavior.
- [ ] New advanced config is typed and validated.

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

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.
