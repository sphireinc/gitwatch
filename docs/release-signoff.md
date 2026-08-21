# Release sign-off record

This is intentionally a fill-in record, not an assertion that release
acceptance has happened. A maintainer should attach command output, recordings,
or CI links for every checked item before moving the launch tasks to
`tasks/completed/`.

## Automated evidence

- [x] `go test ./...`
- [x] `go test -race ./...` where supported
- [x] `go vet ./...`
- [x] performance allocation and benchmark checks
- [x] security checks
- [x] five-target archive extraction and SHA256 verification

## Operator evidence

- [ ] macOS full keyboard/mouse/resize/watch-mode run
- [ ] Linux full keyboard/mouse/resize/watch-mode run
- [ ] Windows full keyboard/mouse/resize/watch-mode run
- [ ] clean-machine install and upgrade
- [ ] Git-missing and non-repository behavior
- [ ] no child Git process or altered terminal state after quit
- [ ] no open P0/P1 issue and no known data-loss issue

## Publication

- [ ] signed release tag
- [ ] GitHub Release and SHA256SUMS
- [ ] installation-channel/package metadata
- [ ] v1.1 issue labels/milestones
- [ ] announcement and demo assets

Signed by:  
Date:  
Release tag:  
