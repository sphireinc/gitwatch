#!/bin/sh
set -eu

out=${1:-dist}
version=${VERSION:-}

test -d "$out"
test -f "$out/SHA256SUMS"
(cd "$out" && shasum -a 256 -c SHA256SUMS)

for target in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64 windows_amd64; do
	archive="$out/gitwatch"
	[ "$target" = windows_amd64 ] && archive="$out/gitwatch"
	if [ -n "$version" ]; then
		file="$out/gitwatch_${version}_${target}.tar.gz"
	else
		file=$(find "$out" -maxdepth 1 -type f -name "gitwatch_*_${target}.tar.gz" -print -quit)
	fi
	test -n "$file"
	test -f "$file"
	dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-artifact.XXXXXX")
	trap 'rm -rf "$dir"' EXIT HUP INT TERM
	tar -xzf "$file" -C "$dir"
	if [ "$target" = windows_amd64 ]; then
		test -f "$dir/gitwatch.exe"
	else
		test -f "$dir/gitwatch"
	fi
	rm -rf "$dir"
	done
echo "release artifacts verified"
