# Task 29 — End-to-end terminal and integration test suite

**Priority:** P0

Build disposable-repository integration tests plus state-machine/component tests. Where practical, use a pseudo-terminal harness for startup/quit/resize/input smoke tests. Keep visual golden tests limited to stable semantic layouts rather than fragile every-character snapshots.

**Acceptance:** CI reliably exercises status → mutate externally → watcher refresh → stage → unstage → quit flows.
