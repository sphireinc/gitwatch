# gitwatch v1.0.0 release notes

gitwatch is an interactive terminal workbench for Git repositories. The v1
release includes live authoritative status, staged/unstaged diff inspection,
safe staging controls, branch and stash management, history and worktree
views, remote operations, a multi-repository dashboard, notifications, and a
versioned out-of-process plugin SDK.

## Safety and compatibility

Git commands use argument vectors and machine-readable output. Destructive
actions require exact confirmations, terminal text is sanitized, plugin output
is bounded, and status refreshes follow mutations. The supported release
targets are macOS amd64/arm64, Linux amd64/arm64, and Windows amd64; Git is an
external runtime dependency.

## Installation

Download the archive for the target platform from the GitHub Release, verify
it against `SHA256SUMS`, place `gitwatch` on `PATH`, and run `gitwatch --help`.
The source-install path remains:

```sh
go install github.com/jusanchez/gitwatch/cmd/gitwatch@v1.0.0
```

Before publication, maintainers must attach the operator validation matrix,
record the supported-terminal evidence, and confirm the signed tag and
installation checks described in `docs/release-checklist.md`.
