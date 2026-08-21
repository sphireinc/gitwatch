# gitwatch v1.0.0 release notes — draft

These notes are a publication draft. v1.0.0 is not released until the operator and publication gates in [the release checklist](release-checklist.md) are complete.

gitwatch is an interactive terminal workbench for Git repositories. The first stable release includes live authoritative status, selected-file staged/unstaged diff inspection, guarded staging, hunk and commit workflows, branch and stash management, history and worktree views, remote operations, a multi-repository dashboard, notifications, optional read-only GitHub visibility, and a versioned out-of-process plugin SDK.

## Safety and compatibility

- Git commands use argument vectors and machine-readable output rather than shell command strings or localized human output.
- Destructive and history-changing actions require exact confirmation.
- Repository, provider, and plugin text is sanitized before terminal display.
- Plugin output, provider responses, history pages, activity, and repository concurrency are bounded.
- Every completed mutation attempt triggers an authoritative Git refresh, including failures and conflict results that may have partially changed repository state.
- Git remains an external runtime dependency; no telemetry is collected.

Release targets are macOS amd64/arm64, Linux amd64/arm64, and Windows amd64.

## Installation

Download the archive for the target platform from the GitHub Release, verify it against `SHA256SUMS`, inspect the included license and dependency notices, place the executable on `PATH`, and run:

```sh
gitwatch --version
gitwatch --help
```

Source installation:

```sh
go install github.com/sphireinc/git-watch/cmd/gitwatch@v1.0.0
```

## Verification

The final release notes must link the accepted operator matrix, CI run, signed tag, checksums, SBOM, provenance, clean-install evidence, and package-manager installation results before publication.
