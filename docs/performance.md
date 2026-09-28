# Performance notes

The status table uses stable row indexes and only materializes visible indexes; it does not allocate a rendered row for every entry on every frame. Git refreshes run outside the Bubble Tea render path and the refresh coordinator permits one status process per repository.

Status-panel row-height measurement is also viewport-bounded. Flat and tree status
views retain the complete logical entry set and stable offsets, while the pure
`internal/ui/virtualrange` model clamps stale offsets and supplies bounded
viewport-plus-overscan candidate ranges. Mouse hit-testing measures only rows
that can occupy the current panel. Rendering accounts for wrapped path rows and
panel headers; table and tree refreshes preserve the selected path and adjust
the offset to keep it visible. The overscan tuning option is
`workspace.status_overscan` (default `4`, range `0`–`32`); the complete Git
snapshot remains in memory regardless of that presentation setting.

Deterministic 14,953-entry regression coverage and benchmarks live in
`internal/app/status_virtualization_test.go`:

```text
go test ./internal/app -run '^$' -bench BenchmarkStatusMouseRowHeights14953 -benchmem
go test ./internal/app -run '^$' -bench '^BenchmarkStatusFileLines14953$' -benchmem
```

The row-height and rendered-file-line benchmarks compare the bounded viewport
with a full-scan baseline. Both are part of `scripts/performance-check.sh`, so
the release performance gate exercises the bounded paths and their comparisons.
On the recorded macOS arm64 / Apple M1 Pro `make check` run, the 14,953-entry
rendered-line sample measured 291.7 microseconds/op, 55.1 KB/op, and 634
allocations/op for the bounded viewport versus 135.1 milliseconds/op, 9.91
MB/op, and 418,721 allocations/op for the full-scan baseline. These are
host-specific observations, not portable latency limits.

The cross-repository command palette has a deterministic 50-repository benchmark:

```text
go test ./internal/app -run '^$' -bench '^BenchmarkCommandPalette50Repositories$' -benchmem -benchtime=1x
```

It indexes only already-loaded in-memory rows and searches the bounded palette
action set; it must not start Git, provider, filesystem, or plugin processes.
The regression test caps the allocation count at 2,500 allocations per search.
An isolated Darwin arm64 / Apple M1 Pro run with Go 1.27.1 in a working tree
based on `fbd07285cfc554c51f740313411cbbd13f24ad7f`, including the new
uncommitted benchmark, measured 35,724 ns/op,
40,122 B/op, and 406 allocs/op with `-benchtime=3s`. This includes palette
action construction and the 50-repository query; it remains host-specific.

`internal/commands/palette_performance_test.go` also measures search over 5,000
already-loaded actions (50 repositories with 100 entries each). It reports a
rare multiword query and the full unfiltered result set separately:

```text
go test ./internal/commands -run '^$' -bench '^BenchmarkSearch5000LoadedActions$' -benchmem -benchtime=3s
go test ./internal/commands -run '^TestSearch5000LoadedActionsAllocationBudget$'
```

The performance check runs both benchmark cases and the allocation regression.
The test caps each case at 1,000 allocations; elapsed time and bytes/op are
reported for comparison, not treated as portable pass/fail limits. On Darwin
arm64 with Go 1.27.1 in a working tree based on
`fbd07285cfc554c51f740313411cbbd13f24ad7f` (including this new uncommitted
benchmark; Apple M1 Pro), the isolated 3-second run measured 428,340 ns/op,
232 B/op, and 4 allocs/op for the rare query, and 357,869 ns/op, 1,842,535
B/op, and 16 allocs/op for the full result set. This package benchmark is non-interactive,
so terminal dimensions are not applicable. These are local measurements, not
Linux or Windows samples or a native-terminal acceptance run.

The scale benchmark and allocation regression cover 1,000, 10,000, and 50,000
changed-path models, with the measured viewport near the end of each list:

```text
go test ./internal/app -run '^$' -bench BenchmarkStatusMouseRowHeightsScale -benchmem
go test ./internal/app -run 'TestStatusMouseRowHeightsScaleAllocationBudget'
```

These are bounded-work regression gates rather than portable latency limits;
the 50,000-entry fixture still requires broader stress, process/goroutine, and
native terminal evidence before Task 183 can be closed.

Record this benchmark with the Go version, OS/architecture, terminal dimensions,
and exact revision. The benchmark includes a full-scan baseline for comparison;
on the recorded Apple M1 Pro run, bounded measurement was approximately 72.8
microseconds/op, 6.2 KB/op, and 287 allocations/op, versus approximately 16.2
milliseconds/op, 1.4 MB/op, and 64,394 allocations/op for the baseline. These
figures are host-specific evidence of bounded measurement work, not a substitute
for native 80x24 keyboard/mouse acceptance.

The repository includes benchmarks for the two critical large-worktree paths:

```text
go test ./internal/git -run '^$' -bench BenchmarkParseStatus10K -benchmem
go test ./internal/ui/table -run '^$' -bench BenchmarkTable10KFilter -benchmem
go test ./internal/git -run TestParseStatus10KAllocationBudget
go test ./internal/patch ./internal/history ./internal/registry \
  -run 'Test(LargePatchAllocationBudget|LargeHistoryAllocationBudgets|RepositoryRowsAllocationBudget)$'
```

The acceptance target is that 10,000 changed paths remain navigable and filtering remains responsive on a supported development machine. Benchmark numbers are machine-dependent; CI treats correctness and bounded behavior as gates, while maintainers should record benchmark output with Go version, OS, CPU, and terminal dimensions when investigating regressions.

Additional workload benchmarks cover `BenchmarkParseLog100K`, `BenchmarkBuildGraph100K`,
`BenchmarkRows1000Repositories`, `BenchmarkParseLargePatch`,
`BenchmarkRefreshInjectedSlowSources`, `BenchmarkRefreshInjectedNetworkLatency`, and
`BenchmarkCapabilityNegotiation`. The refresh benchmarks use context-aware injected
delays at the discovery/snapshot boundaries rather than a live network or filesystem;
this makes the workload deterministic while still exercising the worker pool, timeout,
and cancellation path. Practical budgets for
interactive work are: no Git or filesystem process in `View`, bounded repository refresh
workers, bounded plugin output, and visible-list rendering proportional to the viewport
rather than total history/repository size. Record benchmark output with
`go test -bench . -benchmem` before changing those budgets.

Plugin workspace reloads are single-flight and probe entries serially. Each
plugin child process has a five-second default deadline and cancellation kills
its process tree; this bounds the active plugin-process count to one during a
workspace load even if reload input is repeated.

The registry engine also has a deterministic 100-repository refresh regression,
`TestEngineRefreshKeepsHundredRepositoriesWithinWorkerBound`. It injects a small
cooperative source delay, asserts all 100 repository results are returned in input
order, and proves peak refresh concurrency never exceeds the configured eight
workers. This complements the 20-repository mixed-health isolation scenario and
does not depend on a live network or filesystem.

`TestRefreshCoordinatorBoundsChildrenDuringWatcherEventStorm` joins the
filesystem watcher, refresh coordinator, and a cross-process child counter. A
256-file create burst while the first refresh child is active must yield one
coalesced follow-up refresh; the measured peak child-process count remains one
and returns to zero. This does not model checkout/rebase subprocesses, provider
or history work, or native terminal responsiveness.

CI and the release check enforce allocation budgets for these representative
workloads: fewer than 1,000 allocations for the 10,000-line patch parser,
100,000 for parsing 10,000 porcelain-v2 status entries,
300,000 for parsing 100,000 history records, 200,000 for building the 100,000
node graph, and 100 allocations for producing 1,000 repository rows. These
are structural allocation guards rather than wall-clock limits, so they remain
portable across supported CPUs and operating systems.

The slow-source test also verifies that a repository status operation exceeding
its 15-second budget is cancelled and reported rather than blocking the worker
pool indefinitely.

## Recording a baseline

Record the complete output of the benchmark commands above with `go version`,
OS/architecture, CPU, terminal dimensions, and the exact commit. Compare
allocations and bytes/op first; wall-clock values are useful for a single host
but are not portable release gates. A performance change must include a before
and after record and explain any changed budget.
