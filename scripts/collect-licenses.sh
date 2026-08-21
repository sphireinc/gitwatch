#!/bin/sh
set -eu

output=${1:?usage: collect-licenses.sh OUTPUT_DIRECTORY}
mkdir -p "$output"

modules=$(mktemp "${TMPDIR:-/tmp}/gitwatch-modules.XXXXXX")
trap 'rm -f "$modules"' EXIT HUP INT TERM
go list -m -f '{{if not .Main}}{{.Path}}|{{.Version}}{{end}}' all >"$modules"

while IFS='|' read -r module version; do
	[ -n "$module" ] || continue
	download=$(go mod download -json "$module@$version")
	directory=$(printf '%s\n' "$download" | sed -n 's/^[[:space:]]*"Dir": "\(.*\)",$/\1/p')
	if [ -z "$directory" ]; then
		echo "module source directory unavailable for $module $version" >&2
		exit 1
	fi
	safe=$(printf '%s_%s' "$module" "$version" | tr '/:@' '____')
	found=false
	for source in "$directory"/LICENSE "$directory"/LICENSE.* "$directory"/LICENSE-* \
		"$directory"/COPYING "$directory"/COPYING.* "$directory"/COPYING-* \
		"$directory"/NOTICE "$directory"/NOTICE.* "$directory"/NOTICE-*; do
		if [ -f "$source" ]; then
			filename=${source##*/}
			cp "$source" "$output/${safe}_${filename}"
			found=true
		fi
	done
	if [ "$found" = false ]; then
		echo "no license or notice file found for $module $version" >&2
		exit 1
	fi
done <"$modules"

test -n "$(find "$output" -type f -print -quit)"
