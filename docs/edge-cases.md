# Git edge-case matrix

| Case | Representation/coverage |
|---|---|
| Nested cwd | `rev-parse --show-toplevel` discovery fixture |
| Detached HEAD/unborn/no upstream | discovery and snapshot branch fields |
| Linked worktree and `.git` file | common-dir/git-dir discovery and watcher metadata paths |
| Rename/copy, deletion, mode change | porcelain v2 entry fields and operation argv tests |
| Untracked/ignored/unusual names | NUL parser and leading-hyphen/path tests |
| Merge conflict | unmerged record, conflict type/filter, stage guard |
| Symlink/submodule/very long path | fixture coverage planned for platform CI where supported |

Platform-specific cases that cannot be created reliably on every host remain explicit CI matrix work rather than being silently claimed as locally verified.
