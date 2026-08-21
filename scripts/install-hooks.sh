#!/bin/sh
set -eu

root=$(git rev-parse --show-toplevel)
cd "$root"

git config core.hooksPath .githooks
echo "git hooks enabled from .githooks (pinned Gitleaks fallback: v8.30.1)"
