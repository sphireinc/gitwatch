#!/bin/sh
set -eu

cache=${GOCACHE:-/tmp/git-watch-release-cache}
export GOCACHE="$cache"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

go test ./...
go test -race ./...
go vet ./...
./scripts/performance-check.sh
./scripts/security-check.sh
./scripts/install-check.sh
smoke_binary=$(mktemp "${TMPDIR:-/tmp}/gitwatch-release-smoke.XXXXXX")
demo_dir=
release_dir=
cleanup() {
	[ -z "$demo_dir" ] || rm -rf "$demo_dir"
	[ -z "$release_dir" ] || rm -rf "$release_dir"
	rm -f "$smoke_binary"
}
trap cleanup EXIT HUP INT TERM
go build -o "$smoke_binary" ./cmd/gitwatch
test -n "$("$smoke_binary" --help 2>&1)"
test "$("$smoke_binary" --version | cut -d' ' -f1)" = "1.0.0-dev"
config_output=$("$smoke_binary" --config-inspect)
echo "$config_output" | grep -q '"watch"'

demo_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-demo.XXXXXX")
./scripts/demo-repo.sh "$demo_dir" >/dev/null
status=$(git -C "$demo_dir" status --porcelain=v2 -z --branch)
test -n "$status"

release_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-artifacts.XXXXXX")
VERSION=1.0.0 OUT_DIR="$release_dir" ./scripts/release.sh
VERSION=1.0.0 ./scripts/verify-release-artifacts.sh "$release_dir"
echo "release checks passed"
