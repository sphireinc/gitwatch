#!/bin/sh
set -eu

version=${VERSION:-1.0.0}
version=${version#v}
out=${OUT_DIR:-dist}

case "$version" in
	''|*[!0-9A-Za-z.-]*)
		echo "invalid release version: $version" >&2
		exit 2
		;;
esac

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

commit=${COMMIT:-$(git rev-parse HEAD)}
build_date=${BUILD_DATE:-$(git show -s --format=%cI "$commit")}
work=$(mktemp -d "${TMPDIR:-/tmp}/gitwatch-release.XXXXXX")
trap 'rm -rf "$work"' EXIT HUP INT TERM

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
	if [ "$goos" = windows ]; then
		"$work/releasepack" -format zip -name "$name" -output "$out/$name.zip" -root "$package" -timestamp "$build_date"
	else
		"$work/releasepack" -format tar.gz -name "$name" -output "$out/$name.tar.gz" -root "$package" -timestamp "$build_date"
	fi
done
(cd "$out" && shasum -a 256 ./*.tar.gz ./*.zip >SHA256SUMS)
