# Native terminal harness

The native harness supplements Go tests with reproducible repositories and
human terminal checks. It never treats a screenshot as proof of Git state.

From a clean checkout:

```sh
fixture=$(mktemp -d)
./scripts/native-fixture.sh prepare "$fixture"
./scripts/native-fixture.sh inspect "$fixture"
(cd "$fixture/repository" && gitwatch --watch=fs)
./scripts/native-capture.sh "$fixture/repository" "$fixture/evidence"
./scripts/native-fixture.sh reset "$fixture"
rm -rf "$fixture"
```

For each run record the exact commit, OS/architecture, terminal emulator,
shell, dimensions, encoding, Git version, watch mode, fixture action, and
`PASS`, `FAIL`, or `BLOCKED`. Exercise clean/mixed/conflict/rename/unusual-name
fixtures, resize, keyboard and mouse selection, `NO_COLOR`, reduced/off
motion, cancellation, provider/plugin disabled states, and quit/terminal
restoration. On Windows use PowerShell equivalents and native paths; do not
translate a POSIX result into Windows evidence.

`native-capture.sh` writes only bounded, redacted metadata and porcelain status.
Keep recordings and screenshots outside the repository unless they contain no
personal paths, credentials, private URLs, or repository contents. Delete the
fixture and evidence after reporting, including on failure.

To exercise a linked worktree with the automated terminal harness, pass the
worktree root to `scripts/pty-smoke.sh` or `scripts/record-demo.sh`. These
scripts validate the repository with `git rev-parse`; they do not assume that
`.git` is a directory, because linked worktrees use a `.git` file.
