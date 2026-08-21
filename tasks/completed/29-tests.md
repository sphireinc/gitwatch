# Task 29 — End-to-end terminal and integration test suite

**Priority:** P0

Build disposable-repository integration tests plus state-machine/component tests. Where practical, use a pseudo-terminal harness for startup/quit/resize/input smoke tests. Keep visual golden tests limited to stable semantic layouts rather than fragile every-character snapshots.

**Acceptance:** CI reliably exercises status → mutate externally → watcher refresh → stage → unstage → quit flows.

**Status:** Complete — disposable real-repository integration coverage exercises
initialization, commit, external worktree mutation, authoritative snapshot
refresh, stage, and unstage while the existing app state/component tests cover
deterministic transitions. Native Windows fixtures disable automatic CRLF
conversion, compare normalized paths, avoid filesystem-prohibited path bytes,
and limit POSIX permission assertions to platforms that expose those semantics.
