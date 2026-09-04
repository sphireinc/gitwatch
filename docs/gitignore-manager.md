# `.gitignore` manager domain

The gitignore manager is a repository-scoped composition system over a
user-owned file. `TemplateID` values use the stable upstream relative path
(`root/PHP`, `global/macOS`, or `community/Java/Gradle`), so display-name
changes do not change identity.

## State and ownership

`TemplateMatch.Kind` distinguishes `ManagedExact`, `UnmanagedFull`, `Partial`,
`Absent`, and `InvalidManagedBlock`. A full match is eligible for the UI's
full-match indicator, but only a match with a valid `ManagedBlock` is owned by
gitwatch. Matching pre-existing user text never grants permission to delete
or rewrite it. Ambiguous unmanaged removal must be rejected with a typed
error and presented as a preview for explicit user review.

## Mutation boundary

Every `MutationPlan` carries one repository identity, root, target path, the
complete before-bytes and SHA-256, newline metadata, exact byte edits, selected
template IDs, result bytes, and warnings. Constructors copy all mutable input,
so the plan is a stable preview. The execution layer must re-read the target,
verify the before hash, reject unsafe or symlink targets, and abort with
`ErrConcurrentModification` if the file changed.

The domain does not perform filesystem, Git, network, or TUI work. Repository
adapters provide snapshots and execute plans through typed argument vectors.
After a successful write, the normal watcher may provide a refresh hint, but
the existing repository engine must obtain the authoritative
`git status --porcelain=v2 -z --branch --untracked-files=all` snapshot. The
manager never maintains a second interpretation of worktree status, and all
state is scoped by repository rather than a global active repository.

Managed combinations use the versioned, human-readable block format defined by
`internal/gitignore/managed`: begin metadata records format, stable template
ID, source, upstream commit, and content hash; the end marker repeats the ID.
Unknown versions and crossed IDs are not editable. Duplicate rules across
blocks are intentionally tolerated so each selected template remains exactly
reversible and independently removable.
