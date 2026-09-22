#!/bin/sh
set -eu

cache=${GOCACHE:-/tmp/git-watch-parity-cache}
export GOCACHE="$cache"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

# Keep this gate deterministic and local: all scenarios create disposable
# repositories and use Git's real argv boundary. Provider/network and native
# terminal evidence remain separate release gates.
go test ./internal/integration -run 'Test(RepositoryWorkbenchScenario|MultiRepositoryRefreshTransitionScenario|BatchFetchFiftyDisposableRepositoriesIsBoundedAndFailureIsolated|PathAndCRLFStatusScenarioPreservesGitBytesAndNames|AdvancedHistoryAndComparisonParityScenario|BisectParityScenarioSurvivesFreshLoaderAndReset|CherryPickConflictResumeParityScenario|MergeConflictResumeParityScenario|RebaseConflictResumeParityScenario|RevertConflictResumeParityScenario)$' -count=1
go test ./internal/watch -run 'Test(Watcher|Manager)' -count=1
go test ./internal/bisect ./internal/cherrypick ./internal/submodules -count=1
go test ./internal/remotes ./internal/provider ./internal/multirepo -count=1
go test ./internal/customcmd ./internal/plugins ./pkg/plugin -count=1
go test ./internal/app -run 'TestCustomCommand(PromptFormBlocksExecutionUntilSubmit|ConfirmationCanBeCancelled)$' -count=1
