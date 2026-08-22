# gitwatch task pack

The task files in this directory are the public implementation record. Work
tasks in numeric order unless a dependency exception is documented in the task
itself. Move a task to `tasks/completed/` only after every acceptance criterion
is satisfied and its verification evidence is recorded.

## Current release boundary

Tasks 34, 35, 89, and 90 remain explicitly in progress. Tasks 92 and 102 are
the next release-quality lane; they close evidence and native validation rather
than adding v1 scope. Task 103 may follow once release evidence is complete.

## Planned execution lanes

| Lane | Tasks | Purpose |
| --- | --- | --- |
| v1 release closure | 92, 102, 103 | Candidate evidence, native fixtures, distribution and upgrade verification |
| v1.x UX and reliability | 93, 94, 95, 96, 97, 98, 99, 100, 101, 104 | Compatibility-preserving improvements to interaction, scale, integrations, and supportability |
| v2 preparation | 105 | Freeze contracts and design migrations without pulling breaking behavior into v1.x |

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
