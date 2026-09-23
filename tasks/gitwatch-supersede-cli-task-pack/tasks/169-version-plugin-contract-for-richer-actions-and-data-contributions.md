# Task 169: Version plugin contract for richer actions and data contributions

**Phase:** Extensibility
**Depends on:** 167, 168

## Goal

Extend out-of-process plugins without allowing arbitrary in-process UI code.

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

1. Design plugin protocol vNext with explicit version negotiation and backwards compatibility.
2. Allow plugins to register palette actions, contextual actions, bounded table/detail data, notifications and read-only repository metadata extensions.
3. Host renders schema-defined UI; plugin never injects Go/UI code into process.
4. Declare capabilities for process/network/Git mutation and require config/user approval as appropriate.
5. Keep output/time limits, cancellation and sanitization.
6. Provide dependency-free SDK updates, examples and compatibility fixtures.

## Verification

- Old plugin still works, unknown vNext capability degrades, oversized/control-sequence output bounded.

## Acceptance criteria

- [ ] Plugins are richer while isolation remains a core differentiator.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Progress evidence

- Added an additive API-2 negotiation path in `pkg/plugin`: version lists,
  highest-mutual-version selection, and graceful omission of unknown
  capabilities. The existing strict API-1 handshake remains compatible.
- Added bounded, data-only contribution schemas for contextual actions,
  tables, detail fields, notifications, and repository metadata, plus
  explicit process/network/Git-mutation capability names.
- Added SDK compatibility tests covering API-2 negotiation, unknown-capability
  degradation, oversized contributions, control-sequence rejection, and
  additive API-2 handshake fields.
- Focused verification passed: `go test ./pkg/plugin ./internal/plugins`,
  `go vet ./pkg/plugin ./internal/plugins`, and `git diff --check`.
- Wired the versioned handshake through `internal/plugins.Runtime`: API-2
  manifests advertise `[2, 1]`, hosts accept a negotiated downgrade, and the
  handshake transport can run while optional capabilities are being filtered.
  Ordinary plugin execution still requires every manifest capability to be
  explicitly granted.
- Runtime tests cover API-2 acceptance and unknown-capability degradation;
  focused package, race, vet, and diff checks pass.
- Added the dependency-free `examples/plugin-contribution` API-2 example and
  checked-in `pkg/plugin/testdata/v2` handshake/contribution fixtures.
- Added bounded host-side contribution decoding in `internal/plugins` and
  schema-only contribution summaries in `internal/ui/pluginview`; invalid or
  unknown records are ignored without allowing plugin UI code to enter the
  process.
- Connected enabled plugin reloads to the cancellable runtime probe, attached
  bounded contribution output to discovered entries, and rendered it in the
  plugin workspace. API-2 runtime, manager, UI, full-test, race, vet, and
  diff checks pass.
- Remaining: add provider-backed repository metadata actions and notifications,
  broaden interaction beyond the plugin workspace, and complete
  native/manual/release evidence.

- Schema-defined `notification` contributions are now surfaced through the
  bounded session notification model when an enabled, healthy plugin reloads.
  Titles and descriptions pass the same terminal sanitization boundary as
  other plugin-rendered text; disabled or unhealthy plugins are ignored.
  App coverage records the host notification path. Provider-backed metadata
  actions, broader interaction, and native/manual/release evidence remain
  open.
- At revision `e3bdedb`, the full `make check` gate passed on macOS arm64,
  including plugin contribution/runtime tests, race detection, vet, security
  fuzz checks, formatting, lint, and performance benchmarks. Provider-backed
  metadata actions and native/manual/release evidence remain open.
