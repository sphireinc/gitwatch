#!/bin/sh
set -eu

out=${1:-dist}
version=${VERSION:-}
version=${version#v}

test -d "$out"
test -f "$out/SHA256SUMS"
(cd "$out" && shasum -a 256 -c SHA256SUMS)

for target in darwin_amd64 darwin_arm64 linux_amd64 linux_arm64 windows_amd64; do
	extension=tar.gz
	[ "$target" = windows_amd64 ] && extension=zip
	if [ -n "$version" ]; then
		name="gitwatch_${version}_${target}"
		file="$out/$name.$extension"
	else
		file=$(find "$out" -maxdepth 1 -type f -name "gitwatch_*_${target}.$extension" -print -quit)
		name=$(basename "$file" ".$extension")
	fi
	test -n "$file"
	test -f "$file"
	dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-artifact.XXXXXX")
	if [ "$target" = windows_amd64 ]; then
		unzip -q "$file" -d "$dir"
	else
		tar -xzf "$file" -C "$dir"
	fi
	package="$dir/$name"
	test -f "$package/LICENSE"
	test -f "$package/README.md"
	test -f "$package/THIRD_PARTY_NOTICES.md"
	test -n "$(find "$package/third_party_licenses" -type f -print -quit)"
	if [ "$target" = windows_amd64 ]; then
		test -f "$package/gitwatch.exe"
	else
		test -x "$package/gitwatch"
	fi
	host_target="$(go env GOOS)_$(go env GOARCH)"
	if [ "$target" = "$host_target" ] && [ -n "$version" ]; then
		actual=$("$package/gitwatch" --version)
		case "$actual" in
			"$version ("*) ;;
			*) echo "unexpected version output: $actual" >&2; exit 1 ;;
		esac
	fi
	rm -rf "$dir"
done
echo "release artifacts verified"
