# Task 34 — Pre-v1 beta hardening

**Priority:** P0

Tag `v0.9.0`, distribute binaries, test in real repositories including monorepos and worktrees. Collect bugs specifically around missed refreshes, staging semantics, terminal rendering, Windows behavior, and large repositories. Freeze P0 feature additions after beta; only correctness/polish fixes.

**Acceptance:** Zero open blocker/critical issues and no known data-loss issue.

**Status:** In progress — repository discovery, status rendering, reversible stage/unstage and diff flows, strict five-target artifact verification, repeatable release checks, and partial macOS keyboard evidence are present; the operator matrix in `docs/beta-validation-matrix.md` still requires full macOS, Linux, and native Windows beta evidence.
