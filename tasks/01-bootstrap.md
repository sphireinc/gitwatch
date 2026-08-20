# Task 01 — Bootstrap Go repository and developer tooling

**Priority:** P0

Initialize the module, `cmd/gitwatch`, internal package skeleton from `ARCHITECTURE.md`, Makefile/justfile or equivalent, `.gitignore`, `.editorconfig`, lint config, CI skeleton, version package, and a minimal Bubble Tea v2 full-screen program that restores the terminal correctly on exit. Pin compatible v2 Charm dependencies.

**Acceptance:** `go test ./...`, `go vet ./...`, local run, `--help`, and `--version` work; Ctrl-C and q leave terminal intact.

**Status:** Complete — module, Bubble Tea v2 full-screen shell, package skeleton, build tooling, version flags, and cross-platform CI were added and verified.
