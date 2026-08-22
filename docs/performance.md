# Performance notes

The status table uses stable row indexes and only materializes visible indexes; it does not allocate a rendered row for every entry on every frame. Git refreshes run outside the Bubble Tea render path and the refresh coordinator permits one status process per repository.

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
