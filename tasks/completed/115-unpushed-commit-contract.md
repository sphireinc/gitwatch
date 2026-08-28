# Task 115 — Add a bounded unpushed-commit Git contract

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Tasks 105 and 107

## Objective

Load the commits reachable from the current local branch but not its configured
upstream for a read-only lower-left status pane.

## Requirements

Use argv-only Git execution and machine-readable fields. Detect no upstream,
detached HEAD, unborn branches, shallow history, missing objects, and ahead/behind
state explicitly. Use a bounded equivalent of `git log --graph --decorate
<upstream>..HEAD -n <limit>`, with the default limit of 100 and a hard cap of
1000. Preserve hash, subject, author, date, refs, and graph topology. Include
current HEAD/upstream identity for refresh coalescing.

The operation must be asynchronous, cancellable, generation-safe, sanitized,
and independent of core status refresh. It must never push or mutate Git.

## Acceptance

Exact argument, range, bounds, empty/no-upstream/detached/error, and cancellation
tests pass. The contract exposes enough data for the pane to explain ahead,
behind, and unavailable states without parsing human status output.

**Status:** Complete

## Implementation summary

Implemented `internal/git.LoadUnpushed` with argv-only Git execution, a
100-commit default, 1000-commit hard cap, 256 KiB output bound, upstream-range
counting, sanitized graph output, and explicit no-upstream/error handling.
Added fake-runner coverage for the exact range and bounded request contract.
