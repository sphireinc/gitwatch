# Task 118 — Refresh and cancel context-pane data safely

**Priority:** P2
**Lane:** v1.x reliability
**Dependencies:** Tasks 115 and 116

## Objective

Keep unpushed and branch-summary panes current without blocking status, actions,
shutdown, or repository switching.

## Requirements

Refresh after startup, commits, pulls, pushes, fetches, branch switches,
external commits/ref changes, and authoritative reconciliation. Coalesce or
cancel duplicate requests, reject stale repository generations and request IDs,
and preserve usable prior data on recoverable failures. A successful push must
show zero unpushed commits after the authoritative refresh. No pane refresh may
trigger a mutation.

## Acceptance

Integration, cancellation, race, shutdown, stale-result, external-ref, and
post-push tests pass with no process or goroutine leaks.

**Status:** Complete

## Implementation summary

Context loads are asynchronous, cancelable, bounded, and guarded by repository
generation/request IDs. Authoritative status refreshes update the selected
commit, unpushed, or branch-summary context without mutating Git; repository
switching cancels in-flight work and stale results are discarded.
