#!/bin/sh
set -eu

cache=${GOCACHE:-/tmp/git-watch-performance-cache}
export GOCACHE="$cache"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

go test ./internal/patch ./internal/git ./internal/history ./internal/registry ./internal/ui/table \
	-run 'Test(LargePatchAllocationBudget|ParseStatus10KAllocationBudget|LargeHistoryAllocationBudgets|RepositoryRowsAllocationBudget|TableFilterAllocationBudgetScale)$'
go test ./internal/patch ./internal/git ./internal/history ./internal/registry ./internal/ui/table \
	-run '^$' -bench 'Benchmark(ParseLargePatch|ParseStatus(10K|Scale)|Table(10KFilter|FilterScale)|ParseLog100K|BuildGraph100K|Rows1000Repositories|RefreshInjected(SlowSources|NetworkLatency)|RefreshRepositories)$' \
	-benchmem -benchtime=1x
go test ./internal/app -run '^TestStatusPorcelainSnapshotFilterRender50K(AllocationBudget)?$'
go test ./internal/app -run '^$' -bench '^BenchmarkStatusParseSnapshotFilterRender50K$' \
	-benchmem -benchtime=1x
go test ./internal/app -run '^$' -bench '^BenchmarkStatusMouseRowHeights14953$' \
	-benchmem -benchtime=1x
go test ./internal/app -run '^$' -bench '^BenchmarkStatusFileLines14953$' \
	-benchmem -benchtime=1x
go test ./internal/app -run '^$' -bench '^BenchmarkStatusMouseRowHeightsScale$' \
	-benchmem -benchtime=1x
go test ./internal/app -run '^$' -bench '^BenchmarkCommandPalette50Repositories$' \
	-benchmem -benchtime=1x
go test ./internal/commands -run '^$' -bench '^BenchmarkSearch5000LoadedActions$' \
	-benchmem -benchtime=1x
go test ./internal/commands -run '^TestSearch5000LoadedActionsAllocationBudget$'
