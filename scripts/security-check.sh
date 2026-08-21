#!/bin/sh
set -eu

# Git and plugin processes must receive argument vectors. Keep shell execution
# and interpolated process arguments out of application packages.
if rg -n --glob '*.go' '(sh|bash|powershell)([[:space:]]|",)[^\n]*(-[cC]|/c)|exec\.Command(Context)?\([^\n]*\+' internal cmd pkg; then
	echo "security check failed: shell-string or concatenated process execution found" >&2
	exit 1
fi

GOCACHE=${GOCACHE:-/tmp/git-watch-security-cache} GOPROXY=${GOPROXY:-off} GOSUMDB=${GOSUMDB:-off} go test ./internal/platform ./internal/plugins ./internal/registry
echo "security checks passed"
