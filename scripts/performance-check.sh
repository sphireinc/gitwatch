#!/bin/sh
set -eu

cache=${GOCACHE:-/tmp/git-watch-performance-cache}
export GOCACHE="$cache"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

go test ./internal/patch ./internal/history ./internal/registry \
  -run 'Test(LargePatchAllocationBudget|LargeHistoryAllocationBudgets|RepositoryRowsAllocationBudget)$'
go test ./internal/patch ./internal/history ./internal/registry \
	-run '^$' -bench 'Benchmark(ParseLargePatch|ParseLog100K|BuildGraph100K|Rows1000Repositories|RefreshInjected(SlowSources|NetworkLatency))$' \
	-benchmem -benchtime=1x
