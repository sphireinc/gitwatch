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
test -n "$(go run ./cmd/gitwatch --help 2>&1)"
test "$(go run ./cmd/gitwatch --version | cut -d' ' -f1)" = "0.1.0-dev"
config_output=$(go run ./cmd/gitwatch --config-inspect)
echo "$config_output" | grep -q '"watch"'

demo_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-demo.XXXXXX")
trap 'rm -rf "$demo_dir"' EXIT
./scripts/demo-repo.sh "$demo_dir" >/dev/null
status=$(git -C "$demo_dir" status --porcelain=v2 -z --branch)
test -n "$status"

release_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-artifacts.XXXXXX")
VERSION=0.1.0 OUT_DIR="$release_dir" ./scripts/release.sh
(cd "$release_dir" && shasum -a 256 -c SHA256SUMS)
echo "release checks passed"
