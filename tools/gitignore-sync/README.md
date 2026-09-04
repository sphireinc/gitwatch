# gitignore catalog sync

Generate checked-in offline assets from an immutable upstream commit:

```sh
SOURCE_DATE_EPOCH=0 go run ./tools/gitignore-sync \
  --commit <40-hex-sha> --out internal/gitignore/assets
```

The command uses GitHub's commit-pinned ZIP endpoint by default, rejects unsafe
archive entries, canonicalizes generated template bytes for deterministic
source whitespace, and never fetches `main`. The current upstream snapshot has
three Unix symlink aliases (`Clojure.gitignore`, `Fortran.gitignore`, and
`Global/Octave.gitignore`). The strict default rejects them. A maintainer may
use `--skip-symlinks` only after reviewing the generated diff; skipped aliases
are not dereferenced or represented as template content.

Normal builds and tests read only checked-in assets and do not use the network.
