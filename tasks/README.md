# gitwatch task pack

The task files in this directory are the public implementation record. Work
tasks in numeric order unless a dependency exception is documented in the task
itself. Move a task to `tasks/completed/` only after every acceptance criterion
is satisfied and its verification evidence is recorded.

## Current release boundary

Tasks 34, 35, 89, 90, and 120 remain explicitly in progress. Tasks 34, 35, 89,
and 90 are release and beta evidence lanes; Task 120 is the current v1.x
context-pane release gate. The implementation lanes through Task 119 are
complete and remain in `tasks/completed/` as the public implementation record.
Task 120 remains at the root until its operator-owned native release evidence
is attached. Tasks 121–186 are the Supersede execution lane described in
`tasks/gitwatch-supersede-cli-task-pack/`.

Task 120 remains the prerequisite release/context gate for this lane. Task 121
resets the public roadmap and establishes the parity matrix; Tasks 122–125
then establish the shared repository-scoped sequencer, refresh, and
multi-repository foundations before feature lanes branch.

## Planned execution lanes

| Lane | Tasks | Purpose |
| --- | --- | --- |
| v1 release closure | 92, 102, 103 | Candidate evidence, native fixtures, distribution and upgrade verification |
| v1.x UX and reliability | 93, 94, 95, 96, 97, 98, 99, 100, 101, 104 | Compatibility-preserving improvements to interaction, scale, integrations, and supportability |
| v2 preparation | 105 | Freeze contracts and design migrations without pulling breaking behavior into v1.x |
| v1.x colorized commit tree | 108, 109, 110, 111, 112 | Add safe semantic color rendering to the optional commit graph |
| v1.x commit inspection | 113 | Inspect a selected historical commit and its per-file diffs |
| v1.x context panes | 115, 116, 117, 118, 119, 120 | Track unpushed commits and switch lower-left read-only context panes |
| Supersede foundation | 121, 122, 123, 124, 125 | Reset the roadmap and establish durable, repository-scoped advanced Git foundations |
| Supersede feature and release lanes | 126–186 | Rebase, history operations, conflicts, recovery, integrations, multi-repo differentiation, hardening, and acceptance |

## Dependency shape

```text
34/35/89 ──> 92 ──> 102 ──> 103
                  ├──> 93 ──> 94
                  ├──> 95 ──> 97 ──> 99
                  └──> 96
23/24/82 ───────────────> 98 ─────────┘
70/71/72/86 ────────────> 100 ──> 101
27/28/81 ───────────────────────────> 104
87/98/99/101/103/104 ──────────────> 105
105/107 ──> 108 ──> 109 ──> 110 ──> 111 ──> 112
105/107/111 ───────────────────────────────────> 113
105/107 ──> 115 ──> 116 ──> 117
             └──────> 118 ──> 119 ──> 120
120 ──> 121 ──> 122 ──> 123 ──> 124 ──> 125 ──> 126+
```

Dependencies are planning boundaries, not permission to skip the repository's
normal task gates. Every implementation task must preserve direct argv Git
execution, authoritative refresh after mutations, path-byte fidelity,
sanitized terminal rendering, cancellable background work, keyboard/mouse
parity, and `NO_COLOR`/motion behavior.

## Completion record

Each completed task should record:

- the implementation commit and exact tested revision;
- user-visible behavior, key bindings, configuration, and migration changes;
- focused, integration, race, vet, formatting, lint, security, and performance
  evidence appropriate to the task;
- native/manual evidence separately from automated evidence;
- known limitations, deferred follow-ups, and platform-specific exceptions.

The completed historical task pack remains in `tasks/completed/` for context;
the four in-progress release tasks remain at the directory root until their
operator evidence is complete.
