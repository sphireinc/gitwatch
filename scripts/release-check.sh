#!/bin/sh
set -eu

cache=${GOCACHE:-/tmp/git-watch-release-cache}
candidate_version=${VERSION:-1.0.0}
export GOCACHE="$cache"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

for script in scripts/release.sh scripts/release-check.sh scripts/verify-release-artifacts.sh scripts/verify-sbom.sh; do
	sh -n "$script"
done

go test ./...
go test -race ./...
go vet ./...
./scripts/performance-check.sh
./scripts/security-check.sh
smoke_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-smoke.XXXXXX")
smoke_binary="$smoke_dir/gitwatch"
demo_root=
demo_dir=
release_dir=
sbom_input_dir=
cleanup() {
	[ -z "$demo_root" ] || rm -rf "$demo_root"
	[ -z "$release_dir" ] || rm -rf "$release_dir"
	[ -z "$sbom_input_dir" ] || rm -rf "$sbom_input_dir"
	rm -rf "$smoke_dir"
}
trap cleanup EXIT HUP INT TERM
GOBIN="$smoke_dir" go install ./cmd/gitwatch
test -x "$smoke_binary"
test -n "$("$smoke_binary" --help 2>&1)"
test "$("$smoke_binary" --version | cut -d' ' -f1)" = "1.0.0-dev"
"$smoke_binary" --config-check >/dev/null
config_output=$("$smoke_binary" --config-inspect)
echo "$config_output" | grep -q '"watch"'

demo_root=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-demo.XXXXXX")
demo_dir="$demo_root/repository"
./scripts/demo-repo.sh "$demo_dir" >/dev/null
status=$(git -C "$demo_dir" status --porcelain=v2 -z --branch)
test -n "$status"

release_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-artifacts.XXXXXX")
sbom_input_dir=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-sbom-input.XXXXXX")
VERSION="$candidate_version" OUT_DIR="$release_dir" SBOM_INPUT_DIR="$sbom_input_dir" ./scripts/release.sh
test -x "$sbom_input_dir/gitwatch"
dependency_count=$(go version -m "$sbom_input_dir/gitwatch" | awk '$1 == "dep" { count++ } END { print count + 0 }')
if [ "$dependency_count" -lt 5 ]; then
	echo "release binary exposes only $dependency_count Go dependencies for SBOM generation" >&2
	exit 1
fi
VERSION="$candidate_version" ./scripts/verify-release-artifacts.sh "$release_dir"
echo "release checks passed"
