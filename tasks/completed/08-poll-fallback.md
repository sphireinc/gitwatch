# Task 08 — Polling fallback and reconciliation

**Priority:** P0

Detect watcher initialization/runtime failure and fall back to polling. Add CLI/config `--watch=auto|fs|poll` and `--interval`. In auto mode, use fs events plus low-frequency reconciliation. Show active watch mode in UI.

**Acceptance:** Poll mode works without fsnotify; simulated watcher failure visibly degrades instead of silently going stale.

**Status:** Complete — explicit `auto|fs|poll` mode parsing/selection and bounded polling reconciliation events are implemented and tested, including forced-fs failure and filesystem-change detection.
