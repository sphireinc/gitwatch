# Target capability matrix

Legend:

- **Parity** — required to credibly replace LZ for the workflow.
- **Supersede** — gitwatch should be materially better/different, not merely equal.

| Capability | Target | Tasks |
|---|---|---:|
| Live filesystem-driven authoritative status | **Supersede / preserve** | 124, 183, 184, 186 |
| Multi-repository awareness in all workflows | **Supersede** | 125, 174–178 |
| Interactive rebase | Parity+ | 126–132 |
| Squash/fixup/reword/edit/drop/reorder | Parity+ | 129–131 |
| Fixup commits + autosquash | Parity+ | 130 |
| Cherry-pick single/multi/range | Parity+ | 133–135 |
| Revert durable sequencer workflow | Parity+ | 136 |
| Merge strategies | Parity | 137 |
| Conflict resolution | **Supersede candidate** | 138–143 |
| Reflog browser | Parity | 144 |
| Semantic undo/redo/recovery | **Supersede** | 145–147 |
| Bisect | Parity+ | 148–150 |
| Submodules | Parity+ | 151–154 |
| Tag management/signing | Parity+ | 155–156 |
| Remote CRUD/prune | Parity | 157 |
| Remote branches | Parity | 158–160 |
| Arbitrary revision compare | Parity+ | 161 |
| File history | Parity+ | 162 |
| Blame | **Supersede** | 163 |
| Historical line/hunk editing | Parity | 164 |
| File-tree status mode | Parity | 165 |
| Editor/difftool/mergetool | Parity | 142, 166 |
| Lightweight custom commands | Parity with safer default | 167–168 |
| Isolated richer plugins | **Supersede** | 169 |
| GitHub PR lifecycle | **Supersede** | 170–171 |
| GitHub Actions/check operations | **Supersede** | 172 |
| GitHub issues/releases navigation | Differentiator | 173 |
| Batch multi-repo fetch/pull | **Supersede** | 174 |
| Repository health dashboard | **Supersede** | 175 |
| Operations timeline | **Supersede** | 176 |
| Optional auto-fetch/remote intelligence | **Supersede** | 177 |
| Cross-repo search/palette | **Supersede** | 178 |
| Activity/heat visualization | Differentiator | 179 |
| Actionable scalable graph | Parity+ | 180 |
| Safe migration/config/keymap v3 | Required | 181 |
| Security hardening | Required | 182 |
| Scale/performance | Required | 183 |
| Cross-platform parity evidence | Required | 184 |
| Beta hardening | Required | 185 |
| Stable supersede milestone | Release gate | 186 |

## Required parity before a replacement claim

The project must not claim practical LZ replacement until at least the following are complete and accepted:

- Tasks 126–143 — rebase/cherry-pick/merge/conflicts.
- Tasks 144–150 — reflog/undo/bisect.
- Tasks 151–160 — submodules/tags/remotes/remote branches.
- Tasks 161–169 — compare/history/blame/tree/external/custom extensibility.
- Task 184 — executable parity evidence.

GitHub and multi-repo differentiators are part of the **supersede** claim even if they are not required for basic parity.
