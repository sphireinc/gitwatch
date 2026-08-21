#!/bin/sh
set -eu

mode=${1:---staged}

if ! command -v gitleaks >/dev/null 2>&1; then
	echo "gitleaks is required for secret scanning." >&2
	echo "Install it from https://github.com/gitleaks/gitleaks#installing" >&2
	exit 127
fi

case "$mode" in
	--staged)
	exec gitleaks git --pre-commit --redact --staged --verbose
	;;
	--history)
	exec gitleaks git --redact --verbose
	;;
	*)
	echo "usage: $0 [--staged|--history]" >&2
	exit 2
	;;
esac
