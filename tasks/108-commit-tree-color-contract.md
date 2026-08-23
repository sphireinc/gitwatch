# Task 108 — Define a stable colorized commit-tree Git contract

**Priority:** P2
**Lane:** v1.x UX
**Dependencies:** Tasks 105 and 107

## Objective

Replace the optional commit tree's plain presentation with a bounded,
machine-testable Git log format that carries semantic color roles for the hash,
decorations, subject, relative date, and author while preserving graph topology.

## Requirements

Use Git through an argument vector. The command should be equivalent to:

```text
git --no-pager log --color=always --graph --all --decorate \
  --pretty=format:'%Cred%h%Creset -%C(auto)%d%Creset %s %Cgreen(%cr) %C(bold blue)<%an>%Creset' \
  -n <max_commits>
```

An equivalent stable format is acceptable if it provides the same semantic
fields. It must retain graph characters and merge topology, include abbreviated
hash, decorations, subject, relative date, and author, request color explicitly
because output is captured, and preserve `--all`, cancellation, and the existing
100-commit default/1000-commit ceiling.

Keep the existing byte and line bounds. Define safe behavior for unsupported
Git versions, malformed/truncated color sequences, detached HEAD, unborn
repositories, shallow history, and missing refs. Git color markers are an
intermediate representation, not trusted terminal markup.

## Tests and acceptance

Assert the exact argument vector and fixture output for linear history, merges,
decorated refs, long subjects, Unicode, punctuation-heavy authors, empty
decorations, no commits, output bounds, and cancellation. The loader must return
bounded graph data with identifiable semantic color information, and no raw
repository-controlled escape sequence may reach the final view.

**Status:** Planned
