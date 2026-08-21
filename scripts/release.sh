#!/bin/sh
set -eu

version=${VERSION:-1.0.0}
out=${OUT_DIR:-dist}
rm -rf "$out"
mkdir -p "$out"
for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64; do
	goos=${target%/*}; goarch=${target#*/}; suffix=""; [ "$goos" = windows ] && suffix=".exe"
	name="gitwatch_${version}_${goos}_${goarch}"
	GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/jusanchez/gitwatch/internal/version.Version=$version" -o "$out/gitwatch${suffix}" ./cmd/gitwatch
	(cd "$out" && tar -czf "$name.tar.gz" "gitwatch${suffix}")
	rm "$out/gitwatch${suffix}"
done
(cd "$out" && shasum -a 256 *.tar.gz > SHA256SUMS)
