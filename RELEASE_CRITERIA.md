# v1 Release Criteria

v1.0.0 may be tagged only when:
- `make check`, full-history secret scanning, and the release artifact gate pass on the exact release commit.
- macOS arm64/amd64, Linux amd64/arm64, and Windows amd64 binaries are produced by CI.
- Core watcher/status/stage/unstage flows pass against real temporary Git repos in CI.
- Repository with 10,000 changed paths remains usable; refresh work does not freeze input/render loop.
- Filenames with spaces, unicode, quotes, leading dashes, and renames are safe.
- Merge conflicts are represented correctly and are never accidentally staged/overwritten by unrelated actions.
- Killing/quitting the app leaves no child Git process or altered terminal state.
- No known path can discard work without an explicit confirmation step.
- README includes install, keymap, screenshots/GIF, architecture summary, configuration, troubleshooting, and security notes.
- `gitwatch --version`, `--help`, and graceful non-repository errors work.
- License, changelog, contribution guide, code of conduct, and security policy exist.
- Release archives include the MIT license, third-party notices and license texts, embedded version/commit/date identity, checksums, SBOM, and provenance.
- Native macOS, Linux, and Windows operator evidence is attached for the accepted release commit.
