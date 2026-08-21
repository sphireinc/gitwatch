# Post-v1 Backlog — Explicitly Not Required for v1

The following work is already implemented as post-v1 work in the current
checkout and should not be reintroduced as v1 scope:

- commit composer and amend;
- interactive hunk/line staging;
- stash browser/actions;
- branch switch/create/delete;
- log/graph view;
- remote fetch/pull/push dashboard;
- GitHub hosting integration;
- worktree manager;
- plugin system and SDK;
- custom dashboard extension surfaces;
- multi-repository workspace dashboard.

Future proposals must use `docs/post-v1-triage.md` and remain outside the v1
release commitment. An embedded Git implementation is intentionally not
planned: the system Git executable remains the v1/v2 integration boundary.

These are intentionally deferred so v1 can be deep, safe, and polished around status + inspection + staging.
