# Task 00 — Freeze v1 product contract

**Priority:** P0

Write `docs/product.md` from the supplied product/UX specs. Define personas, primary workflows, non-goals, v1 feature matrix, terminology, and exact safety boundaries. Explicitly define what “htop-style” means operationally: continuously updating dashboard, selection-driven actions, dense status summaries, drill-down detail, event/activity feedback, mouse + keyboard parity.

**Acceptance:** No unresolved P0 product behavior; non-goals include commit authoring, interactive rebase, remote hosting APIs, embedded Git implementation, and plugin system.

**Status:** Complete — `docs/product.md` freezes personas, workflows, terminology, v1 scope, safety boundaries, and explicit non-goals.

**Scope-supersession exception (2026-08-21):** This task captured the original
status-dashboard baseline. Tasks 45–88 subsequently and intentionally expanded
the first stable-release contract to include commit authoring, read-only GitHub
provider views, and an opt-in out-of-process plugin system. Those three original
non-goals are therefore historical and are superseded by the current feature
matrix and safety boundaries in `docs/product.md`. Interactive rebase,
remote-hosting write APIs, and an embedded Git implementation remain non-goals.
Task 00 remains complete as the delivered baseline decision record; this
exception does not constitute beta or release acceptance.
