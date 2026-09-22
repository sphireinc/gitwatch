# Task 182: Perform security hardening for advanced history provider and extensibility

**Phase:** Hardening
**Depends on:** 132, 143, 169, 173, 181

## Goal

Threat-model and harden the larger attack surface before release.

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

1. Update threat model for malicious refs/paths/messages, crafted diffs/conflict markers, `.gitmodules`, remote URLs, provider text, plugin/custom-command output, sequence-editor temp files and external-tool templates.
2. Add architecture/static tests preventing shell command-string execution outside an explicitly reviewed unsafe-shell capability if one ever exists.
3. Fuzz rebase todo, conflict index, reflog, blame, tag, submodule and custom-command parsers.
4. Validate temp-file permissions/cleanup and operation-ID binding for rebase/historical materialization.
5. Ensure provider/plugin/custom-command secrets never enter operation journal/crash diagnostics.
6. Review TOCTOU protection for conflict edits and historical patch editing.
7. Run dependency/vulnerability tooling required by project policy.

## Verification

- Fuzz corpora, terminal-escape regressions, AST/grep execution-boundary checks, race detector.

## Acceptance criteria

- [ ] No feature weakens argv-only default execution or terminal sanitization.
- [ ] Security evidence is recorded with exact tested revision.

## Completion record

- [ ] Implementation commit recorded.
- [ ] Exact tested revision recorded.
- [ ] Focused unit/integration tests recorded.
- [ ] `go test ./...` recorded.
- [ ] Race/vet/lint/format evidence recorded where applicable.
- [ ] Native/manual evidence recorded where this task changes terminal interaction.
- [ ] Known limitations/deferred work documented.

## Current implementation evidence

- Added Go fuzz targets for the rebase todo parser and unmerged-index parser:
  `FuzzParseNeverPanicsOrExceedsInputBound` and
  `FuzzParseIndexNeverPanicsOrExceedsInputBound`.
- Added a 4 MiB input bound to `conflicts.ParseIndex`, matching the existing
  bounded-parser approach used by rebase plans and preventing attacker-sized
  unmerged-index input from creating unbounded parser work.
- Seeded fuzz targets pass as normal tests; extended fuzz-duration evidence and
  the remaining parser/security audit are still required.
- A 2-second rebase fuzz run found 83 new coverage inputs and passed; a
  3-second conflict-index fuzz run found 72 new coverage inputs and passed.
  The conflict fuzz run exposed and the parser now rejects empty-path records;
  the generated corpus is retained under `internal/conflicts/testdata/fuzz`.
- Added `FuzzDefinitionExpandNeverProducesUnsafeArg` for custom-command
  placeholder expansion and argv control-byte rejection. Seeded execution
  passes; extended fuzz-duration evidence remains outstanding.
- Added parser fuzz targets for blame porcelain records and bounded reflog
  records. They assert parser safety and non-negative timestamps/content
  invariants while preserving the existing bounded Git-loading boundaries.
- A 2-second blame fuzz run passed with 78 new coverage inputs; a 2-second
  reflog fuzz run passed with 74 new coverage inputs. The blame corpus includes
  a truncated record with an empty content line, which is valid porcelain for
  a blank source line and is therefore retained without weakening the parser.
- Added fuzz targets for tag-ref parsing and submodule config/status parsing,
  asserting that successful results retain required identity/path invariants.
- Short fuzz runs passed for tags (49 new coverage inputs), submodule config
  (25 new inputs), and submodule status (72 new inputs). These are focused
  parser runs; the complete security gate still requires the broader matrix and
  final exact-revision/native evidence.
- `scripts/security-check.sh` now runs two-second fuzz smoke tests for rebase,
  conflicts, blame, reflog, tags, submodule config/status, and custom-command
  parsing. `GITWATCH_FUZZTIME` can increase the duration for deeper local runs.
- `GOCACHE=/tmp/gitwatch-security-cache GOPROXY=off GOSUMDB=off
  GITWATCH_FUZZTIME=2s ./scripts/security-check.sh` passed. The gate found and
  drove the fix for malformed blame line-count `0`; the parser now retains its
  safe positive default instead of returning `NumLines: 0`.

- The shell-execution invariant now has a portable `grep` fallback when
  `rg` is unavailable, so CI cannot silently skip the argv-boundary scan.

- At revision `7cfef04`, `GOCACHE=/tmp/gitwatch-security-cache GOPROXY=off
  GOSUMDB=off GITWATCH_FUZZTIME=1s ./scripts/security-check.sh` passed all
  parser fuzz smoke lanes, plugin/registry checks, and the shell-execution
  invariant. The full native/manual security review and longer-duration fuzz
  evidence remain open.

- The optional `gh auth token` lookup now uses a bounded 10-second context
  instead of an uncancellable background process, preventing a hung external
  CLI from blocking provider initialization indefinitely. Provider tests and
  the argv/security boundary checks pass; native/manual review and longer
  fuzz-duration evidence remain open.

- At revision `53f5131`, `GOMODCACHE=/tmp/gitwatch-gate-modcache
  GOCACHE=/tmp/gitwatch-gate-cache GITWATCH_FUZZTIME=1s
  ./scripts/security-check.sh` passed the rebase, conflict-index, blame,
  reflog, tag, submodule, custom-command, plugin/registry, and argv-boundary
  security lanes. Native/manual review and longer-duration fuzz evidence
  remain open.

- At revision `8129780`, the same security gate passed with
  `GITWATCH_FUZZTIME=5s`; all parser fuzz lanes completed successfully,
  including rebase, conflict-index, blame, reflog, tags, submodule config and
  status, and custom-command expansion. Native/manual review remains open.
