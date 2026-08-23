#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <version>" >&2
  echo "Example: $0 v1.0.1" >&2
  exit 2
fi

VERSION="$1"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "Error: version must look like v1.0.1" >&2
  exit 2
fi

if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Error: working tree has uncommitted changes." >&2
  exit 1
fi

if git rev-parse "$VERSION" >/dev/null 2>&1; then
  echo "Error: tag $VERSION already exists locally." >&2
  exit 1
fi

if git ls-remote --exit-code --tags origin "refs/tags/$VERSION" >/dev/null 2>&1; then
  echo "Error: tag $VERSION already exists on origin." >&2
  exit 1
fi

echo "Creating signed tag $VERSION..."
git tag -s "$VERSION" -m "gitwatch $VERSION"

echo "Verifying tag signature..."
git tag -v "$VERSION"

echo "Pushing $VERSION to origin..."
git push origin "$VERSION"

#echo "Creating GitHub release..."
#gh release create "$VERSION" \
#  --verify-tag \
#  --generate-notes \
#  --title "gitwatch $VERSION"

#echo "Released gitwatch $VERSION successfully."

echo "Release tag $VERSION pushed."
echo "GitHub Actions will build, attest, and publish the release."
