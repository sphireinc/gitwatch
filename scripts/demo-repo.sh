#!/bin/sh
set -eu

root=${1:-}
if [ -z "$root" ]; then
	root=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-demo.XXXXXX")
elif [ -e "$root" ]; then
	if [ ! -d "$root" ] || [ -n "$(find "$root" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
		echo "refusing to overwrite non-empty demo path: $root" >&2
		exit 2
	fi
	root=$(cd "$root" && pwd -P)
else
	mkdir -p "$root"
fi
git -C "$root" init -q
git -C "$root" config user.name "gitwatch demo"
git -C "$root" config user.email "gitwatch-demo@example.com"
mkdir -p "$root/internal/app" "$root/docs"
printf 'package demo\n\nfunc Ready() bool { return true }\n' > "$root/internal/app/demo.go"
printf '# Demo notes\n\nThis file is intentionally modified for the gitwatch recording.\n' > "$root/docs/notes.md"
git -C "$root" add --all --
git -C "$root" commit -q -m 'demo baseline'
printf 'package demo\n\nfunc Ready() bool { return false }\n' > "$root/internal/app/demo.go"
printf 'untracked demo content\n' > "$root/untracked-notes.txt"
printf '# Demo notes\n\nThis file is intentionally modified for the gitwatch recording.\nA second line demonstrates a larger diff.\n' > "$root/docs/notes.md"
printf 'Demo repository created at %s\n' "$root"
