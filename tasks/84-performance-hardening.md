# Task 84: Benchmark history, multi-repo and patch workflows

Status: In progress

Progress: Added repeatable 100k-history parse/graph, 1k-repository row, and 10k-line patch benchmarks, injected slow-source refresh and plugin capability-overhead benchmarks, documented bounded-render/worker/plugin-output budgets, removed quadratic raw-patch accumulation (10k-line benchmark dropped from about 2 GB to about 6 MB allocations), and added portable allocation-budget tests wired into the release check. Live slow-disk/network/plugin process benchmarks remain deliberately replaced by deterministic injected boundaries.

## Objective
Add repeatable benchmarks for huge diffs, 100k+ commit histories, wide merge graphs, hundreds/thousands of repositories, slow disks, network latency, and plugin overhead. Define CPU/memory/UI-latency budgets and enforce regression thresholds where practical.

## Required implementation
- Produce production-quality implementation, not a prototype.
- Integrate with the existing Bubble Tea message/update architecture and typed Git runner.
- Keep the UI responsive; blocking filesystem, Git, network, and provider work must not run in the render/update hot path.
- Add keyboard and mouse behavior where the task introduces an interactive surface.
- Add structured errors/activity events and refresh affected repository state after mutations.
- Add focused unit/integration tests for success, failure, cancellation, and relevant edge cases.
- Update help/keymap/config/docs when this task adds user-visible behavior.

## Acceptance criteria
- Feature works on macOS, Linux, and Windows unless the task explicitly documents a platform limitation.
- No shell-string interpolation is introduced for Git/process execution.
- User-controlled terminal text is sanitized against control/escape injection.
- Existing v1 status/stage/diff workflows remain functional.
- `go test ./...`, static analysis, and formatting checks pass.
- The task is not complete until automated tests cover its primary behavior.

## Completion artifact
Record implementation notes, key decisions, new commands/keybindings/configuration, tests added, and any deliberately deferred follow-ups in the task/PR completion summary.
