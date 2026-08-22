# Task 97 — Operation lifecycle, cancellation, and queue UX

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 46, 81, 93, and 95

## Objective

Give users a precise, consistent view of asynchronous Git, network, history,
provider, and plugin work, including queued, running, canceled, timed out,
failed, and completed states.

## Required work

- Define a shared operation lifecycle model with stable IDs, repository scope,
  human-safe names, timestamps, progress/indeterminate state, and causes.
- Expose active operations and cancellation in help, notifications, status,
  and workspace views without blocking Bubble Tea update or render paths.
- Define concurrency and conflict policy: which operations queue, which are
  rejected, which supersede prior requests, and which require repository idle.
- Ensure cancellation reaches child contexts and Git/network/plugin processes;
  classify cancellation separately from failure.
- Preserve drafts and user selection after hook, network, provider, or plugin
  failure where continuing is safe.
- Ensure every successful mutation and every state-changing partial operation
  requests authoritative refresh; failures explain whether refresh occurred.

## Acceptance criteria

- Users can see what is running, why an action is unavailable, and how to
  cancel or dismiss it.
- No operation result can update the wrong repository, path, request, or
  generation after cancellation or workspace switching.
- Tests cover queueing, duplicate submission, cancellation races, timeouts,
  shutdown, process cleanup, notification delivery, and refresh ordering.
- Native evidence demonstrates responsive input during slow Git and network
  operations.

## Verification and documentation

Update `docs/advanced-workflows.md`, help text, notifications documentation,
and the architecture concurrency section. Run full, race, leak, and native
shutdown tests.

**Status:** Complete

**Completion summary:** Extended the shared operation engine with stable
lifecycle snapshots, queued/running/completed/failed/canceled/timed-out
classification, causes, timestamps, bounded history, duplicate protection,
repository serialization, and cancellation reporting. Documented concurrency
and refresh semantics; native slow-operation responsiveness remains an
operator-evidence item in the release matrix.
