# Source notes

This task pack was written against the public repository state available on 2026-09-04.

## gitwatch baseline

- https://github.com/sphireinc/gitwatch
- https://github.com/sphireinc/gitwatch/blob/main/ARCHITECTURE.md
- https://github.com/sphireinc/gitwatch/blob/main/ROADMAP.md
- https://github.com/sphireinc/gitwatch/blob/main/KEYMAP.md
- https://github.com/sphireinc/gitwatch/blob/main/docs/advanced-workflows.md
- https://github.com/sphireinc/gitwatch/tree/main/tasks

Important baseline observations used by this pack:

- current architecture explicitly keeps Git/filesystem work out of Bubble Tea render paths;
- `git status --porcelain=v2 -z --branch --untracked-files=all` is the primary status command;
- watcher events are debounced hints with polling/reconciliation fallback;
- Git execution is argv-based;
- the operation engine already provides bounded async work, per-repository scope and cancellation;
- multi-repository registry/plugin/GitHub/history/hunk/stash/branch/worktree/remote foundations already exist;
- the public task record already reaches Task 120;
- the current roadmap explicitly says interactive rebase is not planned, which Task 121 intentionally reverses.

## LZ capability reference

The task pack treats LZ as a capability benchmark, not as an architecture to copy. The benchmark notes cover interactive rebase, keybindings, custom commands, conflict resolution, history recovery, and multi-repository workflows.
