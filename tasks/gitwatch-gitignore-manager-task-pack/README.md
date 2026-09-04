# gitwatch `.gitignore` Manager — Codex Task Pack

This pack continues the supersede-LZ roadmap at **Task 187** and implements a first-class `.gitignore` composition/management system.

## Product intent

The user must be able to:

- open a brand-new Git repository with no `.gitignore`;
- browse GitHub's `github/gitignore` template catalog;
- search immediately (`php`, `node`, `jetbrains`, etc.);
- select multiple template combinations at once;
- create a composed `.gitignore`;
- append combinations later;
- see all full matches pinned at the top with `*`;
- distinguish full, partial, managed, and unmanaged matches;
- remove gitwatch-managed combinations exactly;
- conservatively remove pre-existing combinations without damaging unrelated rules;
- update managed combinations when the catalog changes;
- perform the same work in multi-repository mode;
- remain fully functional offline.

## Architectural position

This feature does **not** replace gitwatch's live status architecture. The `.gitignore` manager modifies repository input. Git remains the authority. After a write, the normal filesystem watcher + canonical `git status --porcelain=v2 -z --branch` pipeline determines what the working tree now looks like.

The feature must never hide/unhide rows by inventing its own interpretation of ignore rules.

## Upstream catalog

Canonical source: https://github.com/github/gitignore

GitHub organizes the catalog into common root templates, `Global/` templates for editors/tools/operating systems, and specialized `community/` templates. The upstream repository is licensed CC0-1.0. gitwatch should embed a deterministic snapshot for offline use and optionally allow a separately validated runtime cache refresh.

## Ownership rule

**gitwatch only owns text inside valid gitwatch managed blocks.**

A pre-existing file can match a template and receive the `*` indicator, but that does not mean gitwatch can freely delete those lines. Unmanaged removal is conservative and previewed. This is deliberately more cautious than treating every matching rule as generated content.

## Task order

Tasks are designed to be executed in numeric order unless dependencies in the current repository make a small reordering unavoidable. Do not start with the TUI. Build the lossless document/mutation/matching core first.
