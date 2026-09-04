# Task 193: Model overlap and rule ownership explicitly

## Objective

Create the overlap/ownership analysis required to add and especially remove combinations without harming rules still needed by other combinations or hand-written content.

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

Many GitHub templates share generic rules. Duplicated ignore rules are semantically harmless, but unsafe deletion is not. Managed blocks have exact ownership. Unmanaged lines do not.

## Implementation steps

1. Build a `RuleOwnershipIndex` for the current plan. For each exact rule line, track occurrences in managed blocks, unmanaged document regions, and every fully/partially matched template that references it.
2. For managed removal, ownership is trivial: remove the selected block span only. Never also delete duplicate unmanaged lines.
3. For unmanaged conservative removal, label a line `safe-to-remove` only if the selected template(s) account for that exact occurrence under the explicit conservative algorithm and the line is not required by an unselected fully matched template. If intent cannot be proven, retain it.
4. Return ambiguity/warning objects explaining shared rules, hand-written duplicates, and why some lines will remain after removing a combination.
5. Expose overlap counts in previews: `PHP and Composer share 3 rules; shared rules will remain` or equivalent.
6. Do not “optimize” managed blocks by globally deduplicating them. That would destroy reversibility.

## Expected code areas

- `internal/gitignore/ownership/*`

## Required tests

- Two managed templates sharing rules.
- Managed + handwritten duplicate.
- Two unmanaged full matches sharing rules.
- Multi-remove of two templates where a third retains a shared rule.

## Acceptance criteria

- [ ] Shared rules are never accidentally removed when another selected/installed template still uses them.
- [ ] Managed removal deletes only its block.
- [ ] Unmanaged ambiguity is visible before mutation.
- [ ] Overlap analysis has deterministic results.

## Definition of done

This task is not done when the UI merely looks correct. It is done only when the behavior is implemented through production code, covered by unit/integration tests appropriate to the task, works under the repository-scoped operation model, and passes `go test ./...` plus the project lint/vet gates.
