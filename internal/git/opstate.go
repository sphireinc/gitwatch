package git

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sphireinc/git-watch/internal/sequencer"
)

const metadataLimit = 1 << 20

// OperationState is the authoritative result of probing one repository for a
// durable Git operation. Found is false when no operation is in progress.
// Diagnostics are bounded, non-authoritative hints for unknown or ambiguous
// Git metadata and are not rendered without normal terminal sanitization.
type OperationState struct {
	Found       bool
	State       sequencer.State
	Diagnostics []string
}

// DetectOperationState reconstructs durable operation state from Git commands
// and the resolved gitdir. It is safe to call after startup or every status
// refresh; it never assumes that gitwatch started the operation. The detector
// reads Git's REBASE_HEAD, CHERRY_PICK_HEAD, REVERT_HEAD, MERGE_HEAD,
// BISECT_GOOD, BISECT_BAD, and HEAD refs through rev-parse; resolves
// rebase-merge, rebase-apply, sequencer, BISECT_START, and ORIG_HEAD through
// rev-parse --git-path; and reads only bounded rebase metadata files beneath
// the resolved rebase directory. No `.git` layout is assumed.
func DetectOperationState(ctx context.Context, discovery Discovery, generation uint64) (OperationState, error) {
	runner := NewRunner(discovery.Root)
	markers, err := operationMarkers(ctx, runner)
	if err != nil {
		return OperationState{}, err
	}
	if len(markers) == 0 {
		return OperationState{}, nil
	}
	if len(markers) != 1 {
		state, stateErr := sequencer.NewState(repositoryID(discovery), generation, sequencer.KindUnknown, sequencer.PhaseActive)
		if stateErr != nil {
			return OperationState{}, stateErr
		}
		return OperationState{Found: true, State: state, Diagnostics: []string{"multiple Git operation markers are active; operation kind is unknown"}}, nil
	}

	marker := markers[0]
	state, err := sequencer.NewState(repositoryID(discovery), generation, marker.kind, sequencer.PhaseActive)
	if err != nil {
		return OperationState{}, err
	}
	state = state.WithHistory(marker.headBefore, sequencer.Recovery{OriginalHead: marker.headBefore, RecoveryRef: marker.recoveryRef}, marker.startedAt)
	state = state.WithObservation(marker.headCurrent, marker.current, marker.remaining, marker.completed, marker.conflicts, marker.updatedAt)
	if marker.kind != sequencer.KindUnknown {
		state, err = state.WithDetails(marker.details)
		if err != nil {
			return OperationState{}, err
		}
	}
	return OperationState{Found: true, State: state, Diagnostics: marker.diagnostics}, nil
}

type operationMarker struct {
	kind        sequencer.Kind
	headBefore  string
	headCurrent string
	current     string
	remaining   int
	completed   int
	conflicts   []string
	recoveryRef string
	startedAt   time.Time
	updatedAt   time.Time
	details     sequencer.Details
	diagnostics []string
}

func operationMarkers(ctx context.Context, runner Runner) ([]operationMarker, error) {
	var markers []operationMarker
	headCurrent, _, err := verifyRef(ctx, runner, "HEAD")
	if err != nil {
		return nil, err
	}
	for _, probe := range []struct {
		name string
		kind sequencer.Kind
	}{
		{"REBASE_HEAD", sequencer.KindRebase},
		{"CHERRY_PICK_HEAD", sequencer.KindCherryPick},
		{"REVERT_HEAD", sequencer.KindRevert},
		{"MERGE_HEAD", sequencer.KindMerge},
	} {
		value, present, err := verifyRef(ctx, runner, probe.name)
		if err != nil {
			return nil, err
		}
		if !present {
			continue
		}
		markers = append(markers, operationMarker{kind: probe.kind, current: value, headCurrent: value, recoveryRef: value})
	}

	paths, err := metadataPaths(ctx, runner, "rebase-merge", "rebase-apply", "sequencer", "BISECT_START", "ORIG_HEAD")
	if err != nil {
		return nil, err
	}
	if paths["rebase-merge"] != "" || paths["rebase-apply"] != "" {
		if hasDirectory(paths["rebase-merge"]) || hasDirectory(paths["rebase-apply"]) {
			marker := readRebaseMarker(paths)
			if replaceKind(&markers, sequencer.KindRebase, marker) {
				// The REBASE_HEAD ref and rebase metadata describe one operation.
			} else {
				markers = append(markers, marker)
			}
		} else {
			// REBASE_HEAD may survive as a transition-only pseudo-ref after Git
			// has removed the rebase state directory. It is not an active
			// operation without the durable rebase metadata.
			markers = removeKind(markers, sequencer.KindRebase)
		}
	}
	if hasDirectory(paths["sequencer"]) && len(markers) == 0 {
		markers = append(markers, operationMarker{kind: sequencer.KindUnknown, diagnostics: []string{"sequencer metadata is present but its operation kind is unknown"}})
	}
	if hasFile(paths["BISECT_START"]) && len(markers) == 0 {
		marker, err := readBisectMarker(ctx, runner, paths)
		if err != nil {
			return nil, err
		}
		markers = append(markers, marker)
	}
	for i := range markers {
		if headCurrent != "" {
			markers[i].headCurrent = headCurrent
		}
		markers[i] = enrichMarker(paths, markers[i])
	}
	return markers, nil
}

func verifyRef(ctx context.Context, runner Runner, ref string) (string, bool, error) {
	result, err := runner.Run(ctx, "rev-parse", "--verify", "-q", ref)
	if err == nil {
		return strings.TrimSpace(string(result.Stdout)), true, nil
	}
	var commandErr *CommandError
	if errors.As(err, &commandErr) && commandErr.Result.ExitCode == 1 {
		return "", false, nil
	}
	return "", false, err
}

func metadataPaths(ctx context.Context, runner Runner, names ...string) (map[string]string, error) {
	paths := make(map[string]string, len(names))
	for _, name := range names {
		result, err := runner.Run(ctx, "rev-parse", "--git-path", name)
		if err != nil {
			return nil, fmt.Errorf("resolve Git metadata path %q: %w", name, err)
		}
		value := strings.TrimSpace(string(result.Stdout))
		if !filepath.IsAbs(value) {
			value = filepath.Join(runner.Dir, value)
		}
		paths[name] = filepath.Clean(value)
	}
	return paths, nil
}

func readRebaseMarker(paths map[string]string) operationMarker {
	base := paths["rebase-merge"]
	if !hasDirectory(base) {
		base = paths["rebase-apply"]
	}
	read := func(name string) string { return readMetadata(filepath.Join(base, name)) }
	marker := operationMarker{kind: sequencer.KindRebase, headBefore: read("orig-head"), recoveryRef: read("onto"), current: read("stopped-sha"), headCurrent: read("head-name")}
	marker.details.Rebase = &sequencer.RebaseDetails{Base: read("head-name"), Onto: read("onto"), Interactive: hasFile(filepath.Join(base, "git-rebase-todo")), TodoRemaining: countMetadataLines(read("git-rebase-todo")), TodoCompleted: countMetadataLines(read("done"))}
	marker.remaining = marker.details.Rebase.TodoRemaining
	marker.completed = marker.details.Rebase.TodoCompleted
	return marker
}

func readBisectMarker(ctx context.Context, runner Runner, paths map[string]string) (operationMarker, error) {
	good, _, err := verifyRef(ctx, runner, "BISECT_GOOD")
	if err != nil {
		return operationMarker{}, err
	}
	bad, _, err := verifyRef(ctx, runner, "BISECT_BAD")
	if err != nil {
		return operationMarker{}, err
	}
	candidate, _, err := verifyRef(ctx, runner, "HEAD")
	if err != nil {
		return operationMarker{}, err
	}
	result, err := runner.Run(ctx, "bisect", "log")
	if err != nil {
		return operationMarker{}, err
	}
	return operationMarker{kind: sequencer.KindBisect, headCurrent: candidate, current: candidate, details: sequencer.Details{Bisect: &sequencer.BisectDetails{Good: good, Bad: bad, Candidate: candidate}}, diagnostics: boundedDiagnostics(string(result.Stdout), paths["BISECT_START"])}, nil
}

func enrichMarker(paths map[string]string, marker operationMarker) operationMarker {
	if marker.headBefore == "" {
		marker.headBefore = readMetadata(paths["ORIG_HEAD"])
	}
	switch marker.kind {
	case sequencer.KindRebase:
		// REBASE_HEAD can remain the only marker during a non-interactive or
		// just-completed transition. Keep the projection valid even when Git's
		// rebase directory no longer exposes interactive details.
		if marker.details.Rebase == nil {
			marker.details.Rebase = &sequencer.RebaseDetails{}
		}
	case sequencer.KindCherryPick:
		commits, completed := readSequencerCommits(paths["sequencer"])
		if len(commits) == 0 {
			commits = nonEmpty(marker.current)
		}
		marker.details.CherryPick = &sequencer.CherryPickDetails{Commits: commits, CurrentIndex: completed}
		marker.completed = completed
		marker.remaining = len(commits) - completed
		if marker.remaining < 0 {
			marker.remaining = 0
		}
	case sequencer.KindRevert:
		marker.details.Revert = &sequencer.RevertDetails{Commits: nonEmpty(marker.current)}
	case sequencer.KindMerge:
		marker.details.Merge = &sequencer.MergeDetails{Other: marker.current, Strategy: "default"}
	}
	return marker
}

func readSequencerCommits(path string) (commits []string, completed int) {
	if !hasDirectory(path) {
		return nil, 0
	}
	for _, name := range []string{"done", "todo"} {
		lines := strings.Split(readMetadata(filepath.Join(path, name)), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 2 || (fields[0] != "pick" && fields[0] != "revert") {
				continue
			}
			commits = append(commits, fields[1])
			if name == "done" {
				completed++
			}
		}
	}
	return commits, completed
}

func nonEmpty(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func repositoryID(discovery Discovery) sequencer.RepositoryID {
	return sequencer.RepositoryID(discovery.Root)
}

func replaceKind(markers *[]operationMarker, kind sequencer.Kind, replacement operationMarker) bool {
	for i := range *markers {
		if (*markers)[i].kind == kind {
			(*markers)[i] = replacement
			return true
		}
	}
	return false
}

func removeKind(markers []operationMarker, kind sequencer.Kind) []operationMarker {
	filtered := markers[:0]
	for _, marker := range markers {
		if marker.kind != kind {
			filtered = append(filtered, marker)
		}
	}
	return filtered
}

func hasFile(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func hasDirectory(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func readMetadata(path string) string {
	if path == "" {
		return ""
	}
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, metadataLimit))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func countMetadataLines(value string) int {
	if value == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(value), "\n"))
}

func boundedDiagnostics(output, path string) []string {
	output = strings.TrimSpace(output)
	if output == "" {
		return nil
	}
	if len(output) > 512 {
		output = output[:512]
	}
	return []string{fmt.Sprintf("git bisect log at %s: %s", path, output)}
}
