#!/bin/sh
set -eu

action=${1:-prepare}
root=${2:-${GITWATCH_FIXTURE_DIR:-/tmp/gitwatch-native-fixture}}
repo="$root/repository"

case "$action" in
prepare)
	rm -rf "$root"
	mkdir -p "$root"
	git init -q "$repo"
	git -C "$repo" config user.name gitwatch-fixture
	git -C "$repo" config user.email fixture@example.invalid
	printf '%s\n' 'baseline' >"$repo/README.md"
	printf '%s\n' 'one' >"$repo/space name.txt"
	git -C "$repo" add --all
	git -C "$repo" commit -q -m baseline
	printf '%s\n' 'changed' >>"$repo/README.md"
	printf '%s\n' 'untracked' >"$repo/untracked file.txt"
	git -C "$repo" add README.md
	printf '%s\n' 'worktree change' >>"$repo/README.md"
	git -C "$repo" mv "space name.txt" "renamed unicode-é.txt"
	printf '%s\n' "fixture=$repo";;
reset)
	test -d "$repo/.git"
	git -C "$repo" reset -q --hard HEAD
	git -C "$repo" clean -q -fd
	printf '%s\n' "reset=$repo";;
inspect)
	test -d "$repo/.git"
	git -C "$repo" --version
	git -C "$repo" status --porcelain=v2 -z --branch | tr '\0' '\n'
	git -C "$repo" worktree list --porcelain;;
*) echo "usage: $0 prepare|reset|inspect [directory]" >&2; exit 2;;
esac
