#!/bin/sh
set -eu

# Git and plugin processes must receive argument vectors. Keep shell execution
# and interpolated process arguments out of application packages.
scan_process_boundaries() {
	if command -v rg >/dev/null 2>&1; then
		rg -n --glob '*.go' '(sh|bash|powershell)([[:space:]]|",)[^\n]*(-[cC]|/c)|exec\.Command(Context)?\([^\n]*\+' internal cmd pkg
		return $?
	fi
	# GitHub's hosted runners do not guarantee ripgrep. Keep the invariant
	# enforced with the ubiquitous POSIX tool instead of silently skipping it.
	grep -R -n -E --include='*.go' '(sh|bash|powershell)([[:space:]]|",)[^\n]*(-[cC]|/c)|exec\.Command(Context)?\([^\n]*\+' internal cmd pkg
}

if scan_process_boundaries; then
	echo "security check failed: shell-string or concatenated process execution found" >&2
	exit 1
fi

export GOCACHE="${GOCACHE:-/tmp/git-watch-security-cache}"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"
go test ./internal/platform ./internal/plugins ./internal/registry

# Run bounded parser fuzz smoke tests in the security gate. Longer fuzzing is
# still available interactively; the gate must remain deterministic enough for
# local release checks while exercising every security-sensitive parser target.
# One second is too close to the baseline-coverage cost on shared CI runners;
# keep the smoke test bounded while leaving enough time for every parser to
# start fuzzing deterministically.
fuzz_time=${GITWATCH_FUZZTIME:-2s}
go test ./internal/rebase -run '^$' -fuzz FuzzParseNeverPanicsOrExceedsInputBound -fuzztime="$fuzz_time"
go test ./internal/conflicts -run '^$' -fuzz FuzzParseIndexNeverPanicsOrExceedsInputBound -fuzztime="$fuzz_time"
go test ./internal/blame -run '^$' -fuzz FuzzParseNeverPanics -fuzztime="$fuzz_time"
go test ./internal/reflog -run '^$' -fuzz FuzzParseNeverPanics -fuzztime="$fuzz_time"
go test ./internal/tags -run '^$' -fuzz FuzzParseTagRefsNeverPanics -fuzztime="$fuzz_time"
go test ./internal/submodules -run '^$' -fuzz FuzzParseConfigNeverPanics -fuzztime="$fuzz_time"
go test ./internal/submodules -run '^$' -fuzz FuzzParseStatusNeverPanics -fuzztime="$fuzz_time"
go test ./internal/customcmd -run '^$' -fuzz FuzzDefinitionExpandNeverProducesUnsafeArg -fuzztime="$fuzz_time"
echo "security checks passed"
