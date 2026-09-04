package assets

import "embed"

// Files contains the checked-in catalog and manifest. It is intentionally
// embedded so normal builds never need network access.
//
//go:embed catalog manifest.json LICENSE.github-gitignore
var Files embed.FS
