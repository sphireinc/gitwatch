#!/bin/sh
set -eu

binary=${1:?usage: pty-large-status-smoke.sh /path/to/gitwatch [evidence-dir]}
evidence=${2:-}
if [ ! -x "$binary" ]; then
	echo "gitwatch binary is not executable: $binary" >&2
	exit 2
fi

root=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-large-status.XXXXXX")
repository="$root/repository"
cleanup() { rm -rf "$root"; }
trap cleanup EXIT HUP INT TERM

mkdir -p "$repository"
git -C "$repository" init --quiet
i=0
while [ "$i" -lt 14953 ]; do
	: >"$repository/status-$i.txt"
	i=$((i + 1))
done

GITWATCH_PTY_ASSERT_LARGE_STATUS=1 ./scripts/pty-smoke.sh "$binary" "$repository" "$evidence"
if [ -n "$evidence" ]; then
	printf 'fixture=14953 untracked files\n' >"$evidence/large-status-fixture.txt"
fi
echo "large-status PTY smoke passed: 14953 untracked files at 80x24"
