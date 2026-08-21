#!/bin/sh
set -eu

root=$(git rev-parse --show-toplevel)
cd "$root"

if ! command -v gitleaks >/dev/null 2>&1; then
	echo "gitleaks is required before enabling the pre-commit hook." >&2
	echo "Install it from https://github.com/gitleaks/gitleaks#installing" >&2
	exit 1
fi

git config core.hooksPath .githooks
echo "git hooks enabled from .githooks"
