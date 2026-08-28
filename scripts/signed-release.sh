#!/bin/sh
set -eu

version=${VERSION:-1.0.6}
version=${version#v}
commit_ref=${COMMIT:-HEAD}
signing_key=${SIGNING_KEY:-}
push=${PUSH:-1}
remote=${REMOTE:-origin}
tag="v$version"

case "$version" in
	''|*[!0-9A-Za-z.-]*)
		echo "invalid release version: $version" >&2
		exit 2
		;;
esac

case "$push" in
	0|1) ;;
	*)
		echo "PUSH must be 0 or 1" >&2
		exit 2
		;;
esac

if ! commit=$(git rev-parse --verify "${commit_ref}^{commit}"); then
	echo "invalid release commit: $commit_ref" >&2
	exit 2
fi

if [ -n "$(git status --porcelain=v1 --untracked-files=all)" ]; then
	echo "refusing to sign a dirty checkout; commit or stash changes first" >&2
	exit 2
fi

if git show-ref --tags --verify --quiet "refs/tags/$tag"; then
	echo "release tag already exists: $tag" >&2
	exit 2
fi

if [ -n "$signing_key" ]; then
	git -c user.signingkey="$signing_key" tag -s "$tag" "$commit" -m "gitwatch $tag"
else
	git tag -s "$tag" "$commit" -m "gitwatch $tag"
fi

if [ "$(git cat-file -t "$tag")" != tag ]; then
	echo "created tag is not annotated: $tag" >&2
	exit 1
fi
git tag -v "$tag" >/dev/null

if [ "$push" = 1 ]; then
	git push "$remote" "$tag"
fi

printf '%s\n' "signed release tag: $tag" "commit: $commit" "remote: $remote" "pushed: $push"
