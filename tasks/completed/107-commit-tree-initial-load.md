# Task 107 — Preserve the initial commit-tree load request

## Status

Complete.

## Objective

Ensure the optional commit tree is populated on the first application startup when enabled through `--with-commit-tree` or configuration.

## Acceptance criteria

- Initial commit-tree results are accepted by the live Bubble Tea model.
- Starting gitwatch in a repository with commits displays the graph instead of incorrectly reporting no commits.
- Initialization does not rely on mutating a value receiver or discard the request identity used by asynchronous results.
- Add regression coverage for the initial load result.
- Run formatting, linting, tests, race tests, vet, diff, security, and performance checks.

## Implementation summary

The initial load now uses a request-preserving initialization command. Later refreshes continue to use the live model request counter, preventing the initial asynchronous result from being rejected as stale.
