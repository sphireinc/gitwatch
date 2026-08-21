# Task 02 — Implement safe Git process runner

**Priority:** P0

Create a context-aware `internal/git.Runner` that executes the system Git binary with argument slices, repository cwd, captured stdout/stderr, exit status, duration, and cancellation. Add structured error types for Git missing, not-a-repository, command failure, cancellation, and unsupported version. Never use a shell.

**Acceptance:** Unit/integration tests prove filenames/args are not shell-interpreted and cancellation terminates child processes.

**Status:** Complete — context-aware argv runner, structured command errors,
captured output/timing, and injection/cancellation tests added. Cancellation is
verified with a child helper that signals successful startup before cancellation,
providing deterministic process-termination coverage across supported platforms
without a shell-dependent command fixture.
