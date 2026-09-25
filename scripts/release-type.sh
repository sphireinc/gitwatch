#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
	echo "Usage: $0 vMAJOR.MINOR.PATCH[-prerelease]" >&2
	exit 2
fi

tag=$1
if ! printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$'; then
	echo "invalid release tag: $tag" >&2
	exit 2
fi

case "$tag" in
	*-*) printf '%s\n' prerelease ;;
	*) printf '%s\n' stable ;;
esac
