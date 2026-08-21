# Beta validation matrix

This file is the evidence sheet for Tasks 34 and 89. Automated checks are
run by CI and `./scripts/release-check.sh`; operator-owned rows must be filled
with the command output or recording link before a public release is called
accepted.

| Area | macOS | Linux | Windows | Evidence required |
| --- | --- | --- | --- | --- |
| Build and launch | observed* | pending | pending | version and help output |
| Real repository refresh | pending | pending | pending | clean, modified, staged, untracked |
| File selection and diff pane | partial* | pending | pending | keyboard and mouse recording |
| Stage/unstage and refresh | partial* | pending | pending | before/after status output |
| Filesystem watch and polling | partial* | pending | pending | external edit refresh output |
| Resize and terminal capability | pending | pending | pending | supported-size recording |
| Git missing / non-repository error | observed* | pending | pending | exit status and message |
| Large repository workload | automated | automated | automated | performance-check output |

Use `./scripts/demo-repo.sh` to create the disposable fixture. Record the
terminal, OS version, terminal emulator, Git version, commit under test, and
whether every row passed. A `pending` row is not a release sign-off.

Partial macOS keyboard-launch/diff evidence is recorded in
[`operator-macos.md`](operator-macos.md); it does not constitute full macOS
release acceptance.

`observed*` and `partial*` refer only to the documented local tmux session;
mouse, resize, watch-mode, clean-install, and other release checks remain
pending.
