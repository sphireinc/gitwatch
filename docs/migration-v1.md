# Migration guide

## Configuration

Configuration without a version, and version-1 configuration files, are
migrated in memory to the version-2 defaults. gitwatch does not rewrite the
source file automatically. Run `gitwatch --config-inspect` to review the
effective configuration, then explicitly update the file if desired.

Version 2 adds repository groups and per-group refresh intervals, remote pull
strategy settings, GitHub/provider settings, plugin directories/output limits,
notification quiet mode, and validated keymap bindings. Unknown future
versions are rejected instead of being silently rewritten.

## Keyboard changes

The core status bindings remain stable. New post-v1 workspaces are opened with
`b` (branches), `n` (remotes), `v` (repositories), and `E` (plugins when
enabled). In the branch workspace, `/` filters, `s` cycles sorting, and
`c`/`R`/`u`/`N`/`D`/`X` create, rename, set/unset upstream, or delete branches.
Use `?` for the context-sensitive keymap.

## Plugins

Plugins are opt-in, out-of-process programs. Existing plugin manifests remain
API-versioned; third-party code should import only `github.com/sphireinc/git-watch/pkg/plugin`.
Review requested capabilities and enablement after upgrading. Runtime output
limits and capability grants continue to apply after migration.
