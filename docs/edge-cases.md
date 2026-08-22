# Git edge-case matrix

| Case | Representation/coverage |
|---|---|
| Nested cwd | `rev-parse --show-toplevel` discovery fixture |
| Detached HEAD/unborn/no upstream | discovery and snapshot branch fields |
| Linked worktree and `.git` file | common-dir/git-dir discovery and watcher metadata paths |
| Rename/copy, deletion, mode change | porcelain v2 entry fields and operation argv tests |
| Untracked/ignored/unusual names | NUL parser and leading-hyphen/path tests |
| Merge conflict | unmerged record, conflict type/filter, stage guard |
| Symlinked discovery metadata | registry and plugin discovery reject symlinked metadata/entries |
| Submodule and very long platform paths | included in native operator acceptance where the host supports creating them |

Platform-specific cases that cannot be created reliably on every host remain explicit CI matrix work rather than being silently claimed as locally verified.
# Git compatibility

gitwatch requires Git 2.23 or newer. This is the first release containing the
machine-readable status contract plus the `restore` and `switch` commands used
by guarded operations. At startup gitwatch records the detected Git version and
derives a session capability set from it. An older or malformed version is
rejected with an actionable diagnostic before the TUI starts.

The core status, refresh, stage, unstage, and diff paths use argument vectors
and porcelain/NUL-delimited output. Optional commands are isolated behind the
capability set; a missing optional capability must disable that action and may
not authorize a destructive fallback. Unknown vendor suffixes are accepted
when the numeric version is parseable. Container/CI compatibility evidence is
separate from native operator evidence.
