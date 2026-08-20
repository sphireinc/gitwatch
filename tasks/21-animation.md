# Task 21 — Animation and delight layer

**Priority:** P1

Add spinner/progress feedback, short-lived row-change highlight, count transitions where practical, clean-state personality, success/error toasts, and subtle watcher heartbeat. Centralize timing. Add `--motion=full|reduced|off`; off mode must eliminate nonessential ticking.

**Acceptance:** Animation never schedules runaway ticks, never changes semantic state, and reduced/off modes are covered by tests.
