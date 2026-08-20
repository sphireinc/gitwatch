# Contributing

Use Go 1.25+ and an installed Git executable. Run `gofmt -w` on changed Go files, then `go test ./...`, `go test -race ./...` where supported, `go vet ./...`, and `git diff --check`.

Keep Bubble Tea models thin, prefer pure transformations, use machine-readable Git output, pass paths as argv elements, and add tests for unusual filenames and cancellation. Include a concise description of user-visible behavior and verification evidence in pull requests.
