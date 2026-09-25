# Configuration

Pass `--config /path/to/config.json` to choose a file explicitly. Otherwise,
gitwatch reads `GITWATCH_CONFIG`, then
`$XDG_CONFIG_HOME/gitwatch/config.json`, then
`$HOME/.config/gitwatch/config.json`. Missing fields are filled from the
version-3 defaults. Files without a version and schema versions 1 or 2 are
accepted and normalized in memory to version 3; the source file is never
rewritten. Future schema versions are rejected so an older binary cannot
silently discard settings. The configuration schema version is independent of
the gitwatch release version. The JSON Schema describes new version-3 files;
legacy files should be checked with the running binary.

The schema-version-3 top-level fields are:

| Field | Default | Purpose |
| --- | --- | --- |
| `version` | `3` | Schema version, independent of the gitwatch release version. |
| `theme` | `auto` | `auto`, `dark`, `light`, or `high-contrast`. |
| `motion` | `full` | `full`, `reduced`, or `off`. |
| `watch` | `auto` | `auto`, `fs`, or `poll`. |
| `interval` | `2s` | Positive base polling interval. |
| `reconciliation` | `30s` | Low-frequency authoritative reconciliation interval. |
| `debounce` | `75ms` | Non-negative filesystem-event debounce. |
| `show_untracked` | `true` | Include untracked paths in status. |
| `show_ignored` | `false` | Include ignored paths where supported. |
| `mouse` | `true` | Enable mouse interaction. |
| `repositories`, `remote`, `github`, `plugins`, `notifications`, `layout`, `diff`, `tools` | See below | Nested configuration objects. |
| `custom_commands` | empty | Optional typed-argv custom commands. |
| `gitignore_max_bytes` | `8388608` | Positive interactive `.gitignore` size limit in bytes (8 MiB). |
| `show_commit_tree` | `false` | Enable the optional commit tree in the Status workspace. |
| `commit_tree` | `max_commits: 100` | Bounded commit-tree history; maximum is `1000`. |
| `workspace` | `200` results, `4` rows | Palette result limit and status-list overscan. |
| `visuals` | enabled, `8` buckets | Dense dashboard indicators and activity history. |
| `profile`, `keymap`, `keymap_profiles` | unset/default | Named keymap selection and safe key overrides; see below. |

Duration fields are Go `time.Duration` values represented as integer
nanoseconds in JSON (for example, `2000000000` is two seconds). This applies to
top-level intervals and duration fields nested under `repositories`, `remote`,
`github`, and `custom_commands`.

### `repositories`

| JSON path | Default | Meaning |
| --- | --- | --- |
| `repositories.roots` | empty | Directories recursively searched for repositories. |
| `repositories.groups` | empty map | Named groups mapped to repository path arrays. |
| `repositories.group_refresh` | empty map | Per-group refresh interval overrides. |
| `repositories.group_auto_fetch` | empty map | Per-group auto-fetch interval overrides. |
| `repositories.ignore_dirs` | empty | Directory names to skip during discovery. Built-in VCS/vendor exclusions also apply. |
| `repositories.max_depth` | `4` | Maximum discovery depth. |
| `repositories.max_repositories` | `256` | Maximum discovered repositories. |

The registry does not follow symlinked directories. A zero discovery depth or
repository limit uses the registry's safe default.

### `remote`

| JSON path | Default | Meaning / constraint |
| --- | --- | --- |
| `remote.pull_strategy` | `ff-only` | Pull strategy: `merge`, `rebase`, or `ff-only`. |
| `remote.stale_after` | `30m` | Non-negative age after which remote state is stale. |
| `remote.workers` | `2` | Bounded concurrent repository workers. |
| `remote.auto_fetch` | `false` | Opt in to background fetch; it never pulls, rebases, or pushes. |
| `remote.auto_fetch_interval` | `30m` | Positive interval required when global auto-fetch is enabled. |
| `remote.auto_fetch_jitter` | `30s` | Non-negative scheduling jitter. |
| `remote.auto_fetch_backoff` | `1m` | Initial retry backoff. |
| `remote.auto_fetch_backoff_max` | `30m` | Maximum retry backoff. |
| `remote.auto_fetch_profiles.<name>` | unset | Optional selected-profile policy object with `enabled`, `interval`, `jitter`, `backoff`, and `backoff_max`; an enabled policy requires a positive interval. |

Auto-fetch pauses when a repository has an active history operation. Fetch
results retain their outcome, completion time, and measured duration for the
repository dashboard.

### `github`, `plugins`, and `notifications`

| JSON path | Default | Meaning |
| --- | --- | --- |
| `github.enabled` | `false` | Enable optional GitHub provider requests. |
| `github.token_env` | `GITHUB_TOKEN` | Name of the environment variable holding the token; the token itself is never stored in config. |
| `github.cache_ttl` | `2m` | Non-negative provider cache lifetime. |
| `plugins.enabled` | `false` | Enable supervised out-of-process plugins. |
| `plugins.directories` | empty | Additional plugin discovery directories. |
| `plugins.max_output` | `1048576` | Non-negative output bound in bytes (1 MiB). |
| `notifications.quiet` | `false` | Suppress attention badges while retaining history. |

### `layout`, `diff`, and `tools`

| JSON path | Default | Meaning / constraint |
| --- | --- | --- |
| `layout.files_percent` | `60` | Wide-layout width assigned to the file panel. |
| `layout.details_percent` | `40` | Wide-layout width assigned to details/diff; both layout percentages must be positive and sum to `100`. |
| `diff.max_bytes` | `4194304` | Positive maximum diff size in bytes (4 MiB). |
| `diff.max_lines` | `20000` | Positive maximum displayed diff lines. |
| `tools.editor`, `tools.opener`, `tools.difftool` | unset | Each object has `executable` and an `args` array. These are argv templates, never shell strings; supported placeholders are `{path}`, `{repo}`, `{left}`, and `{right}`. |

Configuration validation requires both layout percentages to be positive and
their total to equal `100`; totals above or below `100` are rejected.

### `custom_commands`

Each entry requires `name` and `executable`; all other fields are optional:

| Field | Meaning |
| --- | --- |
| `contexts` | Allowed contexts: `status`, `history`, `compare`, `github`, `any`. |
| `binding`, `label` | Optional shortcut and display label. |
| `args` | Argument tokens passed directly to the executable. |
| `directory` | Optional working directory. |
| `timeout` | Non-negative duration. |
| `confirm` | Require confirmation before execution. |
| `mutates_repository` | Declare repository mutation; mutations trigger refresh. |
| `refresh` | Request refresh after completion. |
| `prompts` | Typed form prompts described below. |

Supported argv/directory placeholders are `{repo}`, `{path}`, `{sha}`,
`{branch}`, `{remote}`, `{tag}`, `{url}`, and `{prompt:<id>}`. No shell parsing
or expansion occurs.

Each `prompts[]` object supports `id`, `label`, `kind`, `required`, `pattern`,
`options`, `options_source`, and `default`. `kind` is one of `text`, `secret`,
`confirm`, `select`, or `multi-select`. `select` and `multi-select` require
static `options` or an `options_source` of `branches`, `remotes`, `tags`,
`commits`, or `paths`. `pattern`, when supplied, is a regular expression.
Secret prompt values are redacted from history and diagnostics.

### `commit_tree`, `workspace`, `visuals`, and keymaps

| JSON path | Default | Constraint / meaning |
| --- | --- | --- |
| `commit_tree.max_commits` | `100` | Range `1`–`1000`. |
| `workspace.palette_max_results` | `200` | Range `1`–`2000`. |
| `workspace.status_overscan` | `4` | Range `0`–`32`. |
| `visuals.enabled` | `true` | Enable dense dashboard indicators. |
| `visuals.activity_buckets` | `8` | Range `1`–`32`. |
| `profile` | unset | Select a named `keymap_profiles` entry. |
| `keymap_profiles.<name>` | empty map | Map supported action names to keys for a named profile. |
| `keymap` | built-in defaults | Direct action-to-key overrides; these take precedence over the selected profile. |

Configurable action names are `quit`, `help`, `status`, `branches`, `stashes`,
`history`, `remotes`, `worktrees`, `repositories`, `commit`, `refresh`,
`commit_tree`, `unpushed`, and `branch_summary`. Their built-in bindings are
documented in [the default keymap](../KEYMAP.md). Duplicate keys, unknown
actions, reserved terminal controls, and destructive-action remaps are rejected
before startup.

Validate a file without opening the TUI with:

```text
gitwatch --config-check --config /path/to/config.json
```

For example, make the two wide status panels equal width:

```json
{
  "version": 3,
  "layout": {
    "files_percent": 50,
    "details_percent": 50
  },
  "diff": {
    "max_bytes": 4194304,
    "max_lines": 20000
  }
}
```

The two layout percentages must both be positive and sum to exactly `100`;
totals above or below `100`, zero values, and negative values are configuration
errors.

The following environment variables affect configuration:

| Variable | Effect |
| --- | --- |
| `GITWATCH_CONFIG` | Select the default config-file path (overridden by explicit `--config`). |
| `XDG_CONFIG_HOME` | Sets the config directory used when `GITWATCH_CONFIG` and `--config` are absent. |
| `GITWATCH_PROFILE` | Selects a keymap profile. |
| `GITWATCH_THEME` | Overrides `theme`. |
| `GITWATCH_MOTION` | Overrides `motion`. |
| `GITWATCH_WATCH` | Overrides `watch`. |
| `GITWATCH_INTERVAL` | Overrides `interval`, as a positive integer number of seconds. |
| `GITHUB_TOKEN` | Default GitHub token source when `github.token_env` is unchanged; GitHub CLI authentication may also be used. |

Explicit CLI values take precedence over file and environment values for the
settings they control. Supported configuration-related flags are `--config`,
`--theme`, `--motion`, `--watch`, `--interval`, `--profile`,
`--with-commit-tree`, and `--group`. `--interval` accepts Go duration syntax
(for example, `500ms` or `2s`), while `GITWATCH_INTERVAL` is integer seconds.
Use `--config-check` to validate, `--config-inspect` to print the effective
configuration, and `--config-migration-dry-run` to inspect a migration without
rewriting the source file. `--diagnostics` prints sanitized local diagnostics;
`--support-bundle <path>` writes a sanitized private support bundle.

Keymap precedence is defaults, selected `keymap_profiles.<profile>`, then the
direct `keymap` object. `GITWATCH_PROFILE` and `--profile` select a profile;
the CLI flag wins. Only the documented non-dangerous navigation/workspace
actions are configurable. This keeps restore, delete, discard, and force
operations behind their deliberate built-in confirmations.

The commit tree can be enabled without changing other layout settings:

```json
{
  "show_commit_tree": true,
  "commit_tree": { "max_commits": 100 }
}
```

The CLI flag `--with-commit-tree` also enables it and takes precedence over the
file value. The graph is loaded with a bounded Git log request and refreshes
when HEAD/ref state changes.

The Status workspace also provides built-in lower-left context-pane shortcuts:
`T` for the commit tree, `P` for commits ahead of the configured upstream, and
`B` for a read-only branch summary. These shortcuts work without configuration
and may be overridden by the existing `keymap` or `keymap_profiles` settings.
The lowercase `b` shortcut continues to open the full branch-management view.

The machine-readable schema is available at
[`docs/configuration.schema.json`](configuration.schema.json). Duration values
use the same JSON nanosecond representation as the Go configuration type.

When configured, `Ctrl-E` opens the selected status file in the editor,
`Ctrl-O` opens it with the file opener, and `Ctrl-T` invokes the configured
difftool with `HEAD` and the working-tree path. If no difftool is configured,
`Ctrl-T` uses Git's typed `difftool --no-prompt HEAD -- <path>` command. After
any tool exits, gitwatch requests a normal authoritative status refresh.
