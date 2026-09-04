# gitwatch — Supersede LZ Task Pack

**Tasks:** 121–186
**Prepared:** 2026-09-04
**Execution target:** Codex / agentic implementation, one task at a time
**Objective:** make gitwatch capable of replacing LZ for advanced Git workflows while preserving and strengthening gitwatch's own identity as a live, htop-style, multi-repository Git operations console.

## Why this pack starts at Task 121

The live repository already contains task numbers through 120. This pack therefore starts at **121** to avoid collisions with the public implementation record.

## The product direction

Do **not** turn gitwatch into an LZ clone.

The goal is:

1. Reach practical parity on the advanced workflows that make LZ valuable: interactive rebase, fixup/autosquash, cherry-pick, merge/conflict resolution, reflog recovery, bisect, submodules, tag/remote management, revision comparison, file history/blame, external tools, and lightweight custom commands.
2. Preserve everything gitwatch is already architecturally better positioned to do: live filesystem-driven status, authoritative porcelain-v2 snapshots, safe mutation boundaries, multi-repository operation, bounded background work, capability-bounded plugins, and explicit operation visibility.
3. Supersede rather than imitate by making multi-repository dashboards, repository health, operation timelines, background remote intelligence, cross-repository search, and GitHub workflow visibility first-class.

## Absolute architectural rule

**The live status watcher is never replaced.**

Filesystem events are hints. Git remains truth. The authoritative current-worktree snapshot is still:

```text
git status --porcelain=v2 -z --branch --untracked-files=all
```

Every new feature MUST coexist with this pipeline. Rebase, merge, cherry-pick, conflict resolution, external editors, submodules, provider updates, batch multi-repo work, and plugins may cause more refresh hints, but they do not own Git state.

## Pack contents

- `CODEX_EXECUTION_RULES.md` — mandatory implementation behavior for the coding agent.
- `NON_NEGOTIABLES.md` — architectural invariants that apply to every task.
- `TARGET_CAPABILITY_MATRIX.md` — parity vs differentiation target map.
- `ARCHITECTURE_EXTENSION.md` — opinionated package/domain additions and integration boundaries.
- `DEPENDENCY_GRAPH.md` — recommended execution lanes and dependencies.
- `RELEASE_GATES.md` — what must be true before a stable “supersede” claim.
- `SOURCE_NOTES.md` — current baseline sources used when writing the pack.
- `tasks/121-...md` through `tasks/186-...md` — actionable Codex-ready implementation tasks.

## Recommended execution order

Do not simply parallelize all 66 tasks. The first five establish shared architecture and are prerequisites for almost everything else:

```text
121 → 122 → 123 → 124 → 125
```

After that, the work can branch, but merge/conflict coordination must land before the final lifecycle work for rebase/cherry-pick/revert.

The intended macro-order is:

```text
Foundation
  ↓
Interactive Rebase ───────────────┐
Cherry-pick / Revert ─────────────┤
Merge / Conflict Resolver ────────┘
  ↓
Reflog / Undo / Recovery
  ↓
Bisect + Submodules + Tags/Remotes
  ↓
Comparison / File History / Blame / Tree / External Tools
  ↓
Custom Commands + Plugin vNext
  ↓
GitHub PR / Review / Actions
  ↓
Multi-repo Health / Batch Ops / Timeline / Auto-fetch / Search
  ↓
Hardening / Performance / E2E Parity
  ↓
Beta → Stable Supersede Milestone
```

## Completion discipline

A task is not complete because code compiles.

For each task Codex must:

- implement the domain/process/UI work requested;
- add focused unit tests;
- add integration tests using real disposable Git repositories where Git semantics matter;
- preserve race/cancellation/repository-generation correctness;
- record changes to keymap/config/docs;
- run the repository's normal quality gate;
- record exact evidence in the task's completion section;
- move the task into `tasks/completed/` only after all acceptance criteria are satisfied.

## Definition of “parity”

Do not claim parity merely because gitwatch has buttons with the same names.

Parity means the workflow is usable end-to-end, resumable after restart, safe under external Git activity, tested on real Git repositories, works in narrow terminals, and remains compatible with watcher-driven status and multi-repository state.

## Definition of “supersede”

The project may claim the supersede milestone only when:

- all required parity rows are green;
- live filesystem-driven status remains the default/core behavior;
- advanced Git workflows are durable/recoverable;
- multi-repository operation is first-class rather than a separate mode bolted on later;
- operation history/health makes Git behavior more observable than a traditional Git TUI;
- no P0/P1 data-loss, stale-state, cross-repository, security, or sequencer-recovery defects remain.
