# Task 08 — Polling fallback and reconciliation

**Priority:** P0

Detect watcher initialization/runtime failure and fall back to polling. Add CLI/config `--watch=auto|fs|poll` and `--interval`. In auto mode, use fs events plus low-frequency reconciliation. Show active watch mode in UI.

**Acceptance:** Poll mode works without fsnotify; simulated watcher failure visibly degrades instead of silently going stale.

**Status:** Complete — explicit `auto|fs|poll` mode parsing/selection, bounded polling that excludes Git object payloads, low-frequency filesystem reconciliation, visible setup/runtime fallback, active-mode rendering, and authoritative refresh integration are implemented and tested.
