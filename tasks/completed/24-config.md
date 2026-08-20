# Task 24 — Configuration and CLI contract

**Priority:** P0

Implement precedence: built-in defaults < config file < environment where explicitly supported < CLI flags. Configure theme, motion, watch mode, polling interval, reconciliation interval, untracked visibility, ignored visibility, mouse, refresh debounce, key overrides only if robust. Use XDG-style locations with platform-appropriate fallbacks.

**Acceptance:** `gitwatch config` or documented config inspection makes effective settings debuggable; invalid config fails clearly.

**Status:** Complete — typed defaults, XDG/GITWATCH_CONFIG path resolution, JSON loading, environment and CLI precedence, validation, and effective-config inspection serialization are implemented and tested.
