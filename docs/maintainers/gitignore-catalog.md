# Maintaining the gitignore catalog

The checked-in catalog is an offline, deterministic data snapshot from the
commit-pinned `github/gitignore` repository. It is data only: gitwatch does not
execute template content, hooks, scripts, or archive entries.

## Syncing a new upstream commit

Review the upstream commit and its archive before changing the pin. Generate a
candidate catalog with a fixed timestamp:

```sh
SOURCE_DATE_EPOCH=0 go run ./tools/gitignore-sync \
  --commit <40-hex-upstream-commit> \
  --out internal/gitignore/assets
```

The importer accepts only regular files, rejects traversal, backslash and
absolute archive names, rejects symlinks unless `--skip-symlinks` is explicitly
reviewed, requires the upstream `LICENSE`, and bounds archive and entry sizes.
It classifies only the supported root, `Global/`, and `community/` template
paths. Never substitute a branch or moving URL for the commit pin.

## Review and verification

Inspect the complete generated diff, especially template additions/removals,
line-ending changes, source paths, and license content. Verify the manifest and
all content hashes through the catalog loader, then run:

```sh
go test ./internal/gitignore/...
go test ./internal/integration -run Gitignore
go test -race ./internal/gitignore/...
go vet ./internal/gitignore/...
git diff --check
```

The manifest stores the upstream repository, exact commit, stable template ID,
source path, byte count, and SHA-256 content hash. `catalog.Default` validates
the embedded manifest and files at load time; the runtime cache uses the same
validation and falls back to the embedded snapshot on any cache error. Keep
the generated license and manifest changes in the same focused commit as the
catalog data.

Do not hand-edit generated catalog files to resolve a review disagreement.
Change the source commit or importer behavior, regenerate with a fixed
`SOURCE_DATE_EPOCH`, and review the resulting deterministic diff again.
