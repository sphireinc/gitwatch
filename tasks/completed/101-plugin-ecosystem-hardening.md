# Task 101 — Plugin ecosystem hardening and SDK evolution policy

**Priority:** P1
**Lane:** v1.x compatibility-preserving
**Dependencies:** Tasks 73–75, 87, 93, 97, and 100

## Objective

Harden the out-of-process plugin host and make the public SDK practical for
third-party authors without weakening capability isolation or API compatibility.

## Required work

- Freeze and document API-1 wire fixtures, handshake errors, capability grants,
  message limits, timeouts, process limits, and shutdown semantics.
- Test malformed, oversized, slow, crashing, repeatedly restarting, and hostile
  plugin output; ensure no terminal escape injection or unbounded allocation.
- Define plugin discovery precedence, trust/enable state, version display,
  disabled reasons, reload behavior, and state-file permissions.
- Add SDK helpers/examples for commands, panels, widgets, status inputs, and
  structured errors without coupling the SDK to Bubble Tea internals.
- Document API compatibility policy, deprecation windows, capability review,
  security reporting, and reproducible example builds.
- Ensure plugin work cannot block status refresh, mutation confirmation,
  shutdown, or repository switching.

## Acceptance criteria

- API-1 compatibility fixtures pass host and SDK tests independently.
- Every limit and capability is enforced at the process boundary and tested.
- A hostile plugin cannot alter terminal state, read ungranted data, hang quit,
  or starve other plugins/core UI.
- Public examples build on supported platforms and explain expected output.
- No plugin API break is introduced without a version bump and migration notes.

## Verification and documentation

Run fuzz/budget/race tests, process-leak checks, security scans, example builds,
and native enable/disable/crash evidence. Update `docs/plugin-sdk.md`, plugin
contract docs, threat model, and release notes.

**Status:** Complete

**Completion summary:** Added public SDK message builders and structured error
payloads, documented API-1 additive evolution and capability/limit policy, and
retained host-side bounded output, timeout, cancellation, restart, handshake,
and state-permission enforcement with compatibility tests and examples.
