# Performance notes

The status table uses stable row indexes and only materializes visible indexes; it does not allocate a rendered row for every entry on every frame. Git refreshes run outside the Bubble Tea render path and the refresh coordinator permits one status process per repository.

The repository includes benchmarks for the two critical large-worktree paths:

```text
go test ./internal/git -run '^$' -bench BenchmarkParseStatus10K -benchmem
go test ./internal/ui/table -run '^$' -bench BenchmarkTable10KFilter -benchmem
```

The acceptance target is that 10,000 changed paths remain navigable and filtering remains responsive on a supported development machine. Benchmark numbers are machine-dependent; CI treats correctness and bounded behavior as gates, while maintainers should record benchmark output with Go version, OS, CPU, and terminal dimensions when investigating regressions.
