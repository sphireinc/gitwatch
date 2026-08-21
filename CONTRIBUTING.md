# Contributing

Thank you for helping improve gitwatch. Bug reports, documentation fixes, tests, portability work, and focused feature proposals are welcome.

## Before opening an issue

- Search existing issues and the [roadmap](ROADMAP.md).
- Use the security process in [SECURITY.md](SECURITY.md) for suspected command injection, terminal injection, credential exposure, or data-loss vulnerabilities.
- Include the gitwatch version or commit, OS/architecture, terminal and size, Git version, watch mode, reproduction steps, and sanitized output.
- Never attach credentials, private repository contents, remote URLs containing secrets, or unredacted debug logs.

## Development setup

Use Go 1.25.10 and Git for the documented contributor gate. `make check` invokes the pinned golangci-lint v2.12.0 module through `go run`, so an unversioned global linter cannot silently change the result. The `go 1.25.0` directive in `go.mod` is the module's language-version floor; `.go-version` is the contributor-toolchain pin. The canonical module is `github.com/sphireinc/git-watch`.

Confirm the active tools before running the gate:

```sh
go version                  # must report go1.25.10
cat .golangci-lint-version # must report v2.12.0
```

```sh
git clone git@github.com:sphireinc/git-watch.git
cd git-watch
make install-hooks
make check
```

The checked-in pre-commit hook runs formatting/diff hygiene, a staged secret scan, and a full-history scan. It uses Gitleaks from `PATH` when present or runs the pinned v8.30.1 module through Go. See `scripts/install-hooks.sh` for setup; the same full-history gate runs in public CI.

## Implementation expectations

- Read [AGENTS.md](AGENTS.md), [ARCHITECTURE.md](ARCHITECTURE.md), and [UX_SPEC.md](UX_SPEC.md) before changing behavior.
- Keep Bubble Tea models thin and move business rules into pure, testable packages.
- Use machine-readable and NUL-delimited Git output when available.
- Invoke Git with argument slices; never construct a shell command string.
- Preserve unusual paths and test spaces, tabs, Unicode, quotes, leading hyphens, renames, and conflicts where relevant.
- Keep filesystem, Git, network, provider, history, and plugin work outside the render path and context-cancellable.
- Require explicit confirmation for destructive or history-changing actions.
- Trigger an authoritative refresh after every mutation attempt completes, including failure or conflict results that may have changed the worktree or index.
- Preserve keyboard/mouse parity, `NO_COLOR`, high-contrast-safe semantics, and reduced/off motion.
- Document new user-visible behavior and update the keymap/configuration contract when applicable.

## Verification

Run the complete local gate before requesting review:

```sh
make check
```

That target checks formatting, lint, tests, the race detector, vet, Git diff hygiene, security boundaries, and representative performance budgets. Platform-specific behavior should include operator evidence when it cannot be validated automatically.

## Pull requests

Keep pull requests focused and include:

- the user-visible outcome and motivation;
- safety and compatibility considerations;
- tests and documentation added or updated;
- exact verification commands and results; and
- any platform behavior that remains unverified.

Contributions are submitted under the repository's [MIT License](LICENSE). By contributing, you confirm that you have the right to submit the work under those terms.
