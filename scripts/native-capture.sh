#!/bin/sh
set -eu

fixture=${1:?fixture repository path required}
output=${2:?evidence output directory required}
mkdir -p "$output"
{
	date -u '+%Y-%m-%dT%H:%M:%SZ'
	git --version
	uname -srm 2>/dev/null || true
	git -C "$fixture" status --porcelain=v2 --branch
	git -C "$fixture" worktree list --porcelain
} | sed -E 's#(/Users|/home|[A-Za-z]:\\Users)[^[:space:]]*#<redacted-path>#g; s#(https?://)[^[:space:]]+@#\1<redacted>@#g' >"$output/fixture.txt"
printf '%s\n' 'Record terminal, emulator, dimensions, watch mode, exact commit, and PASS/FAIL/BLOCKED separately.' >"$output/README.txt"
