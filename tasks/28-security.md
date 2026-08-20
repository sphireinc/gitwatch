# Task 28 — Security and data-loss audit

**Priority:** P0

Threat-model command injection, malicious filenames/control sequences, terminal escape injection from Git paths/errors, symlink/path confusion, destructive command scope, config file trust, subprocess lifecycle. Escape/sanitize terminal control bytes before rendering untrusted path/stderr content.

**Acceptance:** Add regression tests for ANSI/control characters and shell metacharacters in filenames. No known command injection or silent data-loss path.
