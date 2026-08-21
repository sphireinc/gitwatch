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
go build ./cmd/gitwatch
smoke_binary=$(mktemp "${TMPDIR:-/tmp}/gitwatch-release-smoke.XXXXXX")
trap 'rm -rf "$demo_dir" "$smoke_binary"' EXIT
go build -o "$smoke_binary" ./cmd/gitwatch
test -n "$("$smoke_binary" --help 2>&1)"
test "$("$smoke_binary" --version | cut -d' ' -f1)" = "0.1.0-dev"
config_output=$("$smoke_binary" --config-inspect)
echo "$config_output" | grep -q '"watch"'

demo_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-demo.XXXXXX")
./scripts/demo-repo.sh "$demo_dir" >/dev/null
status=$(git -C "$demo_dir" status --porcelain=v2 -z --branch)
test -n "$status"

release_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-artifacts.XXXXXX")
VERSION=0.1.0 OUT_DIR="$release_dir" ./scripts/release.sh
VERSION=0.1.0 ./scripts/verify-release-artifacts.sh "$release_dir"
echo "release checks passed"
