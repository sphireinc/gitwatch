# Supersede release gates

## P0 blockers

Any one of these blocks the stable milestone:

- data loss caused by gitwatch;
- stale status requiring manual refresh for correctness;
- repository A operation/state applied to repository B;
- unrecoverable rebase/cherry-pick/revert/merge state after gitwatch restart;
- shell/argument injection from repository-controlled data;
- terminal control-sequence injection;
- watcher/process/goroutine runaway under normal documented scale;
- conflict resolver overwriting a file changed externally after the view was loaded;
- unsafe automatic undo after repository state diverges;
- destructive operation without explicit scope-specific confirmation.

## Core product gate

The following statement must still be true:

> gitwatch continuously reflects authoritative Git status from live filesystem hints with polling/reconciliation fallback.

If advanced work causes a regression where gitwatch mostly refreshes after its own commands, the release fails even if every new feature works.

## Multi-repo gate

At minimum prove:

- 50 registered repositories remain bounded;
- mixed healthy/broken repositories remain isolated;
- operation/sequencer attention is visible from dashboard;
- unrelated repositories can refresh while one repo is rebasing/conflicted;
- batch fetch/pull respects worker limits/cancellation;
- switching repositories drops late generation results correctly.

## Advanced Git parity gate

Must be end-to-end and restart-safe:

- interactive rebase;
- squash/fixup/reword/edit/drop/reorder;
- fixup/autosquash;
- cherry-pick;
- merge + unified conflicts;
- reflog recovery + safe undo;
- bisect;
- submodules;
- tags/remotes/remote branches;
- revision compare;
- file history/blame;
- tree view;
- external tools;
- custom commands.

## UX gate

Every new workspace:

- works at 80x24;
- supports keyboard and mouse parity;
- supports NO_COLOR;
- supports reduced/off motion;
- has context-sensitive help/command palette entry;
- displays loading/error/empty states;
- remains cancellable/navigable during background work.

## Security gate

- argv-only default process execution;
- parser fuzz/regression evidence;
- sanitized terminal rendering;
- remote/provider/plugin/custom-command secret redaction;
- private temporary files;
- no generic hard-reset/raw-force/clean shortcuts.

## Performance gate

Document and meet project-specific budgets for:

- 1k/10k/50k changed paths;
- 10/50/100 registered repositories;
- event storms during checkout/rebase;
- large histories/reflogs/blame files;
- provider refresh under multi-repo mode;
- batch operations.

## Claim gate

Do not say “gitwatch supersedes LZ” in README/release copy until `PARITY_MATRIX.md` is fully evidenced for required rows.
