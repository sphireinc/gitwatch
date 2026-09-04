# Non-negotiable architecture and product invariants

These rules apply to **every task in this pack**. Codex should treat a proposed implementation that violates them as incorrect even if it appears to satisfy a local feature requirement.

## 1. Watcher-first identity never goes away

`internal/watch` and the refresh coordinator remain the heart of gitwatch.

- fsnotify/filesystem events request refresh.
- polling remains fallback/reconciliation.
- manual refresh remains an escape hatch.
- no workspace may require manual refresh for correctness.
- advanced operations must not disable watchers to simplify their implementation.

## 2. Git status is authoritative

The canonical live worktree snapshot remains porcelain v2 + NUL-delimited output. Feature-local optimistic state can be used for transient UX, but it MUST reconcile to authoritative Git state immediately.

After every mutation:

1. mutation exits/pauses;
2. request authoritative refresh;
3. parse immutable repository state;
4. derive UI state from that snapshot;
5. never pretend the mutation succeeded merely because a button was pressed.

## 3. External Git activity is a supported workflow

The user may run Git in another terminal while gitwatch is open.

The application must detect and recover from:

- external `git add`/unstage;
- branch checkout;
- rebase/merge/cherry-pick/revert/bisect start;
- conflict resolution;
- continue/abort;
- ref changes;
- editor changes;
- commit/amend.

A gitwatch operation is not special. Git state is special.

## 4. Multi-repo is not optional architecture

Every new operation/domain object carries repository identity.

- No global `currentRebase` singleton.
- No global cherry-pick selection without repo scope.
- No unbounded background worker per repo.
- One repository failure never blocks another repository's summary.
- Late async results from repo A never land in repo B after a workspace switch.
- Repository dashboards surface active sequencer/conflict/provider attention.

## 5. Process execution stays typed and argv-based

Never build shell strings from repository data.

Use:

```go
exec.CommandContext(ctx, "git", "diff", "--", path)
```

Do not use:

```go
exec.CommandContext(ctx, "sh", "-c", "git diff -- " + path)
```

Custom commands default to executable + argv. Any future unsafe shell mode must be separately named, explicit, warned, and disabled by default.

## 6. Git is not reimplemented

Use Git plumbing/porcelain and parse machine-readable output. Do not write an embedded object database, merge engine, rebase implementation, ref database, or custom index writer unless no supported Git mechanism exists and a separate architecture decision approves it.

## 7. Destructive actions stay deliberate

Keep the existing safety posture:

- no generic `git reset --hard` shortcut;
- no raw `git push --force` shortcut;
- no `git clean -fd` shortcut;
- exact path/ref/range confirmation for destructive/history-rewriting actions;
- published-history warnings before rebase/fixup/history patch edits;
- stale-state checks before undo or conflict-file writes.

## 8. Bubble Tea owns UI state

No Git/network/filesystem/process call in `View()` or synchronous render/update code that can block the event loop.

All expensive work must be:

- cancellable;
- bounded;
- generation-scoped;
- returned as typed messages.

## 9. Every list and stream is bounded

Bound or virtualize:

- changed files;
- diffs;
- history;
- reflog;
- blame;
- operation timeline;
- provider responses;
- plugin/custom-command output;
- repositories;
- submodules;
- tags;
- remote branches;
- queues/workers/goroutines/processes.

## 10. Terminal text is hostile input

Sanitize before rendering:

- paths;
- branch/ref names;
- commit subjects/authors;
- remote URLs;
- conflict text;
- provider comments/titles;
- plugin/custom-command output;
- Git diagnostics.

Preserve raw bytes separately where exact Git/path behavior requires them.

## 11. Accessibility/current terminal behavior remains supported

Every new workspace must:

- work at 80x24;
- support keyboard and mouse parity;
- honor `NO_COLOR`;
- not encode status solely by color;
- honor full/reduced/off motion;
- remain display-width aware.

## 12. Existing contracts are respected

Current plugin/config compatibility cannot be casually broken. If a breaking boundary is necessary:

- bump schema/protocol version;
- provide migration;
- add compatibility fixtures;
- document upgrade and rollback.
