#!/bin/sh
set -eu

sbom=${1:?usage: verify-sbom.sh path/to/sbom.spdx.json}

test -f "$sbom"
if ! command -v jq >/dev/null 2>&1; then
	echo "jq is required to verify the release SBOM" >&2
	exit 1
fi

if ! jq -e '
	((.name // "") | ascii_downcase | contains("gitwatch")) or
	any(.packages[]?;
		((.name // "") | ascii_downcase | contains("gitwatch")) or
		any(.externalRefs[]?;
			(.referenceLocator // "") | contains("github.com/sphireinc/git-watch")
		)
	)
' "$sbom" >/dev/null; then
	echo "SBOM does not identify the gitwatch release component" >&2
	exit 1
fi

dependency_count=$(jq '[
	.packages[]? |
	select(any(.externalRefs[]?;
		((.referenceType // "") | ascii_downcase) == "purl" and
		((.referenceLocator // "") | startswith("pkg:golang/")) and
		((.referenceLocator // "") | contains("github.com/sphireinc/git-watch") | not)
	)) |
	.SPDXID
] | unique | length' "$sbom")
if [ "$dependency_count" -lt 5 ]; then
	echo "SBOM contains only $dependency_count external Go dependencies; expected at least 5" >&2
	exit 1
fi

if ! jq -e 'any(.packages[]?;
	any(.externalRefs[]?;
		(.referenceLocator // "") | contains("charm.land/bubbletea")
	)
)' "$sbom" >/dev/null; then
	echo "SBOM does not contain the linked Bubble Tea dependency" >&2
	exit 1
fi

echo "release SBOM verified ($dependency_count external Go dependencies)"
