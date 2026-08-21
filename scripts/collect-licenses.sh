#!/bin/sh
set -eu

output=${1:?usage: collect-licenses.sh OUTPUT_DIRECTORY}
mkdir -p "$output"
go mod download all

modules=$(mktemp "${TMPDIR:-/tmp}/gitwatch-modules.XXXXXX")
trap 'rm -f "$modules"' EXIT HUP INT TERM
go list -m -f '{{if not .Main}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}' all >"$modules"

while IFS='|' read -r module version directory; do
	[ -n "$module" ] || continue
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
