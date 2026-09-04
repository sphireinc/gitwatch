# Task 140: Build unified conflict resolver workspace

**Phase:** Merge and conflicts
**Depends on:** 138, 139, 125

## Goal

Create a dedicated htop-style conflict workspace used by merge, rebase, cherry-pick and revert.

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

1. Create `internal/ui/conflictview` with conflict list plus selected conflict detail in wide mode and equivalent stacked/overlay mode at 80x24.
2. Show operation kind, branch/target, total/resolved count and selected path.
3. Add previous/next conflict and previous/next hunk navigation.
4. Allow jumping to live Status without abandoning conflict context.
5. Expose whole-file ours/theirs, both where valid text, edit externally, mark resolved/stage, restore unresolved and continue-operation actions.
6. Mouse hit targets for resolution must require deliberate clicks; do not make one-click destructive choices easy.
7. Resolution/staged indicators come from the current authoritative index snapshot.

## UI/UX requirements

- Use textual Ours/Theirs/Both labels in addition to color.
- Keep active sequencer status visible in header/footer.
- Esc navigates back; it never aborts the Git operation.

## Verification

- Wide/80x24 snapshots, keyboard/mouse parity, external resolution while open.

## Acceptance criteria

- [ ] A multi-file conflict can be resolved without leaving gitwatch.
- [ ] UI state follows index/status rather than optimistic assumptions.

## Completion record

- [x] Implementation commits recorded: conflict workspace foundation, app routing, typed whole-file actions, external tool/lifecycle actions, mouse parity, and bounded detail loading.
- [x] Exact tested revision recorded: final implementation revision is recorded by the completion commit below.
- [x] Focused unit/integration tests recorded: `go test ./internal/git ./internal/app ./internal/ui/conflictview`, including a real two-file merge-conflict resolution fixture.
- [x] `go test ./...` recorded: passed.
- [x] Race/vet/lint/format evidence recorded where applicable: `go test -race ./...`, `go vet ./...`, `gofmt`, and `git diff --check` passed; pinned `make check` reaches lint but golangci-lint is blocked by the environment's denied Go build-cache path.
- [x] Native/manual evidence recorded where this task changes terminal interaction: automated wide/80x24 rendering and keyboard/mouse parity tests passed; interactive human terminal QA remains deferred and is documented as a release-gate limitation.
- [x] Known limitations/deferred work documented: unified cross-operation coordinator and hunk-level/manual edit semantics remain in Tasks 141 and 143; the Both action restores Git's two-sided merge presentation without implicitly staging it.
