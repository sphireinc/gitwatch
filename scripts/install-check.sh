#!/bin/sh
set -eu

cache=${GOCACHE:-/tmp/git-watch-install-cache}
export GOCACHE="$cache"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

bin_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-install-bin.XXXXXX")
demo_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-install-demo.XXXXXX")
trap 'rm -rf "$bin_dir" "$demo_dir"' EXIT HUP INT TERM

GOBIN="$bin_dir" go install ./cmd/gitwatch
test -x "$bin_dir/gitwatch"
test "$("$bin_dir/gitwatch" --version | cut -d' ' -f1)" = "1.0.0-dev"
test -n "$("$bin_dir/gitwatch" --help 2>&1)"
"$bin_dir/gitwatch" --config-check >/dev/null

./scripts/demo-repo.sh "$demo_dir" >/dev/null
status=$(git -C "$demo_dir" status --porcelain=v2 -z --branch)
test -n "$status"
echo "clean install checks passed"
