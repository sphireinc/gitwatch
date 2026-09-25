#!/bin/sh
set -eu

test "$(./scripts/release-type.sh v1.2.3)" = stable
test "$(./scripts/release-type.sh v1.2.3-beta.1)" = prerelease
test "$(./scripts/release-type.sh v2.0.0-rc.1)" = prerelease

if ./scripts/release-type.sh 1.2.3 >/dev/null 2>&1; then
	echo "release type accepted a tag without the v prefix" >&2
	exit 1
fi
if ./scripts/release-type.sh v1.2.x-beta >/dev/null 2>&1; then
	echo "release type accepted a non-numeric stable version component" >&2
	exit 1
fi

workflow=.github/workflows/release.yml
grep -Fq 'release_type=$(./scripts/release-type.sh "$GITHUB_REF_NAME")' "$workflow"
grep -Fq 'release_flags+=(--prerelease)' "$workflow"
grep -Fq 'gh release edit "$GITHUB_REF_NAME" --title "gitwatch ${GITHUB_REF_NAME}" "${release_flags[@]}"' "$workflow"
if ! awk '/gh release create/{in_create=1} in_create && /"\$\{release_flags\[@\]\}"/{found=1} END{exit !found}' "$workflow"; then
	echo "release creation does not apply the channel-specific flags" >&2
	exit 1
fi

echo "release channel policy checks passed"
