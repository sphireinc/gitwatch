# Task 104 — Diagnostics, support bundles, and no-telemetry observability

**Priority:** P2
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 27, 28, 81, 92, and 97

## Objective

Help users and maintainers diagnose failures locally without collecting
telemetry, leaking repository content, or turning debug logging into a hidden
data export.

## Required work

- Define structured diagnostic categories for Git, watcher, config, terminal,
  provider, plugin, operation, and shutdown failures.
- Add an explicit user-invoked diagnostic view/command showing versions,
  capabilities, modes, bounded timings, and redacted state.
- Provide an opt-in support bundle that contains sanitized metadata and logs,
  never file contents, tokens, full remote URLs with credentials, or arbitrary
  plugin/provider payloads.
- Add correlation IDs for related operation/refresh/error messages without
  using them as user tracking identifiers.
- Make diagnostics useful in colorless, non-interactive, startup-failure, and
  terminal-too-small contexts.
- Document retention, permissions, deletion, redaction limits, and how users
  can inspect a bundle before sharing it.

## Acceptance criteria

- A user can obtain actionable diagnostics after startup, Git, refresh,
  provider, plugin, and shutdown failures without restarting repeatedly.
- Redaction tests cover paths, refs, URLs, headers, tokens, control bytes, and
  nested error causes; support bundles are bounded in size.
- Diagnostics never invoke telemetry, phone home, or alter normal operation.
- Support output is stable enough for issue templates but does not promise a
  machine-parseable API unless explicitly versioned.

## Verification and documentation

Add unit/security tests, update SECURITY and troubleshooting docs, and provide
an issue-template section requesting sanitized diagnostic output.

**Status:** Complete

**Completion summary:** Added explicit local diagnostics and support-bundle
commands, stable metadata schema/correlation IDs, bounded private atomic JSON
writes, redaction/control-byte/path sanitization tests, troubleshooting and
security guidance, and a sanitized diagnostics section in the public issue
template. No telemetry or network activity is introduced.
