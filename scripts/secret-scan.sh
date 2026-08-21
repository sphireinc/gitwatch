#!/bin/sh
set -eu

mode=${1:---staged}
gitleaks_version=v8.30.1

run_gitleaks() {
	if command -v gitleaks >/dev/null 2>&1; then
		gitleaks "$@"
		return
	fi
	go run "github.com/zricethezav/gitleaks/v8@${gitleaks_version}" "$@"
}

case "$mode" in
	--staged)
	run_gitleaks git --pre-commit --redact --staged --verbose
	;;
	--history)
	run_gitleaks git --redact --verbose
	;;
	*)
	echo "usage: $0 [--staged|--history]" >&2
	exit 2
	;;
esac
