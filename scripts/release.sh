#!/bin/sh
set -eu

version=${VERSION:-1.0.0}
version=${version#v}
out=${OUT_DIR:-dist}
allow_dirty=${ALLOW_DIRTY_SOURCE:-0}

case "$version" in
	''|*[!0-9A-Za-z.-]*)
		echo "invalid release version: $version" >&2
		exit 2
		;;
esac

case "$allow_dirty" in
	0|1) ;;
	*)
		echo "ALLOW_DIRTY_SOURCE must be 0 or 1" >&2
		exit 2
		;;
esac

head_commit=$(git rev-parse --verify 'HEAD^{commit}')
requested_commit=${COMMIT:-$head_commit}
if ! commit=$(git rev-parse --verify "${requested_commit}^{commit}"); then
	echo "invalid release commit: $requested_commit" >&2
	exit 2
fi
if [ "$commit" != "$head_commit" ]; then
	echo "release commit $commit does not match checked-out HEAD $head_commit" >&2
	exit 2
fi
if [ "$allow_dirty" != 1 ] && [ -n "$(git status --porcelain=v1 --untracked-files=all)" ]; then
	echo "refusing to package a dirty checkout; commit or stash changes first" >&2
	exit 2
fi

if [ -e "$out" ]; then
	test -d "$out"
	if [ -n "$(find "$out" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
		echo "refusing to overwrite non-empty output directory: $out" >&2
		exit 2
	fi
else
	mkdir -p "$out"
fi
out=$(cd "$out" && pwd)

build_date=${BUILD_DATE:-$(git show -s --format=%cI "$commit")}
sbom_input=${SBOM_INPUT_DIR:-}
remove_sbom_input=0
if [ -z "$sbom_input" ]; then
	sbom_input=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release-sbom.XXXXXX")
	remove_sbom_input=1
fi
if [ -e "$sbom_input" ]; then
	test -d "$sbom_input"
	if [ -n "$(find "$sbom_input" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
		echo "refusing to overwrite non-empty SBOM input directory: $sbom_input" >&2
		exit 2
	fi
else
	mkdir -p "$sbom_input"
fi
sbom_input=$(cd "$sbom_input" && pwd)
if [ "$sbom_input" = "$out" ]; then
	echo "SBOM input directory must not be the release output directory" >&2
	exit 2
fi
work=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release.XXXXXX")
cleanup() {
	rm -rf "$work"
	[ "$remove_sbom_input" = 0 ] || rm -rf "$sbom_input"
}
trap cleanup EXIT HUP INT TERM

common="$work/common"
mkdir -p "$common/third_party_licenses"
cp LICENSE README.md THIRD_PARTY_NOTICES.md "$common/"
./scripts/collect-licenses.sh "$common/third_party_licenses"
go build -buildvcs=false -trimpath -o "$work/releasepack" ./internal/releasepackcmd

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64; do
	goos=${target%/*}
	goarch=${target#*/}
	suffix=""
	[ "$goos" = windows ] && suffix=".exe"
	name="gitwatch_${version}_${goos}_${goarch}"
	package="$work/$name"
	cp -R "$common" "$package"
	GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -buildvcs=false -trimpath \
		-ldflags "-s -w -X github.com/sphireinc/git-watch/internal/version.Version=$version -X github.com/sphireinc/git-watch/internal/version.Commit=$commit -X github.com/sphireinc/git-watch/internal/version.BuildDate=$build_date" \
		-o "$package/gitwatch${suffix}" ./cmd/gitwatch
	if [ "$target" = linux/amd64 ]; then
		cp "$package/gitwatch" "$sbom_input/gitwatch"
	fi
	if [ "$goos" = windows ]; then
		"$work/releasepack" -format zip -name "$name" -output "$out/$name.zip" -root "$package" -timestamp "$build_date"
	else
		"$work/releasepack" -format tar.gz -name "$name" -output "$out/$name.tar.gz" -root "$package" -timestamp "$build_date"
	fi
done
metadata="$out/gitwatch_${version}_release.json"
{
	printf '{\n'
	printf '  "schema": 1,\n'
	printf '  "name": "gitwatch",\n'
	printf '  "version": "%s",\n' "$version"
	printf '  "commit": "%s",\n' "$commit"
	printf '  "build_date": "%s",\n' "$build_date"
	printf '  "repository": "https://github.com/sphireinc/gitwatch",\n'
	printf '  "license": "MIT",\n'
	printf '  "targets": ["darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64", "windows/amd64"]\n'
	printf '}\n'
} >"$metadata"
(cd "$out" && shasum -a 256 ./*.tar.gz ./*.zip ./*_release.json >SHA256SUMS)
