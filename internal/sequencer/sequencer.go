// Package sequencer models durable, repository-scoped Git operation state.
//
// Git metadata and the authoritative porcelain-v2 status snapshot remain the
// source of truth. A State is an immutable projection reconstructed from Git;
// it is not persistence and it does not execute commands. The operation engine
// owns scheduling and cancellation separately from this package.
package sequencer

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// RepositoryID identifies the repository whose Git state was observed.
type RepositoryID string

// Kind identifies the durable Git operation in progress.
type Kind uint8

const (
	KindUnknown Kind = iota
	KindRebase
	KindCherryPick
	KindRevert
	KindMerge
	KindBisect
)

func (k Kind) String() string {
	switch k {
	case KindRebase:
		return "rebase"
	case KindCherryPick:
		return "cherry-pick"
	case KindRevert:
		return "revert"
	case KindMerge:
		return "merge"
	case KindBisect:
		return "bisect"
	default:
		return "unknown"
	}
}

// Phase is the lifecycle phase reconstructed from Git metadata.
type Phase uint8

const (
	PhaseUnknown Phase = iota
	PhaseActive
	PhasePaused
	PhaseCompleted
	PhaseAborted
	PhaseFailed
)

func (p Phase) String() string {
	switch p {
	case PhaseActive:
		return "active"
	case PhasePaused:
		return "paused"
	case PhaseCompleted:
		return "completed"
	case PhaseAborted:
		return "aborted"
	case PhaseFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Recovery contains bounded recovery information associated with a projection.
type Recovery struct {
	OriginalHead string
	RecoveryRef  string
}

// RebaseDetails describes a rebase without embedding Git's on-disk layout.
type RebaseDetails struct {
	Base          string
	Onto          string
	Interactive   bool
	TodoRemaining int
	TodoCompleted int
}

// CherryPickDetails describes the selected commits and current position.
type CherryPickDetails struct {
	Commits      []string
	CurrentIndex int
	Mainline     int
}

// RevertDetails describes an ordered revert sequence.
type RevertDetails struct {
	Commits      []string
	CurrentIndex int
}

// MergeDetails describes the merge target and selected strategy.
type MergeDetails struct {
	Other    string
	Strategy string
}

// BisectDetails describes the current bisect boundaries and candidate.
type BisectDetails struct {
	Good      string
	Bad       string
	Candidate string
	Steps     int
}

// Details is a typed union; only the member matching State.Kind is populated.
type Details struct {
	Rebase     *RebaseDetails
	CherryPick *CherryPickDetails
	Revert     *RevertDetails
	Merge      *MergeDetails
	Bisect     *BisectDetails
}

// State is an immutable projection of one repository's durable Git operation.
// Use NewState and With methods to construct derived values; slices returned by
// accessors are always copies.
type State struct {
	repositoryID RepositoryID
	generation   uint64
	kind         Kind
	phase        Phase
	headBefore   string
	headCurrent  string
	target       string
	current      string
	remaining    int
	completed    int
	conflicts    []string
	startedAt    time.Time
	updatedAt    time.Time
	recovery     Recovery
	details      Details
}

// NewState constructs a validated projection. Unknown operations are allowed
// so newer Git metadata can be surfaced safely as an explicit unknown state.
func NewState(repositoryID RepositoryID, generation uint64, kind Kind, phase Phase) (State, error) {
	if repositoryID == "" {
		return State{}, errors.New("sequencer repository ID is required")
	}
	if kind < KindUnknown || kind > KindBisect {
		return State{}, fmt.Errorf("invalid sequencer kind %d", kind)
	}
	if phase < PhaseUnknown || phase > PhaseFailed {
		return State{}, fmt.Errorf("invalid sequencer phase %d", phase)
	}
	return State{repositoryID: repositoryID, generation: generation, kind: kind, phase: phase}, nil
}

// RepositoryID returns the repository scope of the projection.
func (s State) RepositoryID() RepositoryID { return s.repositoryID }

// Generation returns the refresh generation that produced the projection.
func (s State) Generation() uint64 { return s.generation }

// Kind returns the operation kind.
func (s State) Kind() Kind { return s.kind }

// Phase returns the lifecycle phase.
func (s State) Phase() Phase { return s.phase }

// HeadBefore returns the HEAD identity captured before the operation.
func (s State) HeadBefore() string { return s.headBefore }

// HeadCurrent returns the current HEAD identity.
func (s State) HeadCurrent() string { return s.headCurrent }

// Target returns the operation target or range description.
func (s State) Target() string { return s.target }

// CurrentCommit returns the commit currently being processed.
func (s State) CurrentCommit() string { return s.current }

// Remaining returns the bounded number of remaining steps, when Git exposes it.
func (s State) Remaining() int { return s.remaining }

// Completed returns the bounded number of completed steps.
func (s State) Completed() int { return s.completed }

// ConflictPaths returns an owned, sorted copy of conflict paths.
func (s State) ConflictPaths() []string { return append([]string(nil), s.conflicts...) }

// StartedAt returns the observed start timestamp.
func (s State) StartedAt() time.Time { return s.startedAt }

// UpdatedAt returns the latest observation timestamp.
func (s State) UpdatedAt() time.Time { return s.updatedAt }

// Recovery returns recovery metadata as a value.
func (s State) Recovery() Recovery { return s.recovery }

// Details returns a deep copy of typed operation-specific details.
func (s State) Details() Details { return cloneDetails(s.details) }

// WithObservation returns a copy with the latest Git-derived observation.
func (s State) WithObservation(headCurrent, current string, remaining, completed int, conflicts []string, observedAt time.Time) State {
	s.headCurrent = headCurrent
	s.current = current
	s.remaining = maxZero(remaining)
	s.completed = maxZero(completed)
	s.conflicts = uniqueSorted(conflicts)
	s.updatedAt = observedAt
	return s
}

// WithHistory returns a copy with pre-operation and recovery identities.
func (s State) WithHistory(headBefore string, recovery Recovery, startedAt time.Time) State {
	s.headBefore = headBefore
	s.recovery = recovery
	s.startedAt = startedAt
	return s
}

// WithDetails returns a copy with validated typed operation details.
func (s State) WithDetails(details Details) (State, error) {
	if err := validateDetails(s.kind, details); err != nil {
		return State{}, err
	}
	s.details = cloneDetails(details)
	return s, nil
}

// Event describes an observable lifecycle transition from Git.
type Event uint8

const (
	EventObserveActive Event = iota
	EventPause
	EventResume
	EventComplete
	EventAbort
	EventFail
)

func (e Event) String() string {
	switch e {
	case EventObserveActive:
		return "observe active"
	case EventPause:
		return "pause"
	case EventResume:
		return "resume"
	case EventComplete:
		return "complete"
	case EventAbort:
		return "abort"
	case EventFail:
		return "fail"
	default:
		return "unknown"
	}
}

// Transition applies one valid Git-observed lifecycle transition.
func Transition(state State, event Event, observedAt time.Time) (State, error) {
	if event < EventObserveActive || event > EventFail {
		return State{}, fmt.Errorf("invalid sequencer event %d", event)
	}
	next := state
	switch state.phase {
	case PhaseUnknown:
		if event != EventObserveActive {
			return State{}, invalidTransition(state.phase, event)
		}
		next.phase = PhaseActive
	case PhaseActive:
		switch event {
		case EventObserveActive:
		case EventPause:
			next.phase = PhasePaused
		case EventComplete:
			next.phase = PhaseCompleted
		case EventAbort:
			next.phase = PhaseAborted
		case EventFail:
			next.phase = PhaseFailed
		default:
			return State{}, invalidTransition(state.phase, event)
		}
	case PhasePaused:
		switch event {
		case EventPause:
		case EventResume, EventObserveActive:
			next.phase = PhaseActive
		case EventAbort:
			next.phase = PhaseAborted
		case EventFail:
			next.phase = PhaseFailed
		default:
			return State{}, invalidTransition(state.phase, event)
		}
	default:
		return State{}, invalidTransition(state.phase, event)
	}
	next.updatedAt = observedAt
	return next, nil
}

func invalidTransition(phase Phase, event Event) error {
	return fmt.Errorf("invalid sequencer transition: %s -> %s", phase, event)
}

// Message carries a projection through an asynchronous refresh boundary.
type Message struct {
	RepositoryID RepositoryID
	Generation   uint64
	State        State
}

// ApplyMessage accepts a message only for the active repository and generation.
// A zero active generation is accepted for callers initializing a workspace.
func ApplyMessage(current State, message Message, activeRepository RepositoryID, activeGeneration uint64) (State, error) {
	if message.RepositoryID == "" || message.RepositoryID != activeRepository {
		return State{}, errors.New("sequencer message belongs to a different repository")
	}
	if message.State.RepositoryID() != message.RepositoryID || message.State.Generation() != message.Generation {
		return State{}, errors.New("sequencer message scope does not match its state")
	}
	if activeGeneration != 0 && message.Generation < activeGeneration {
		return State{}, errors.New("stale sequencer message generation")
	}
	if current.RepositoryID() != "" && current.RepositoryID() != message.RepositoryID {
		return State{}, errors.New("sequencer state belongs to a different repository")
	}
	return message.State, nil
}

func validateDetails(kind Kind, details Details) error {
	switch kind {
	case KindRebase:
		if details.Rebase == nil {
			return errors.New("rebase details are required")
		}
	case KindCherryPick:
		if details.CherryPick == nil {
			return errors.New("cherry-pick details are required")
		}
	case KindRevert:
		if details.Revert == nil {
			return errors.New("revert details are required")
		}
	case KindMerge:
		if details.Merge == nil {
			return errors.New("merge details are required")
		}
	case KindBisect:
		if details.Bisect == nil {
			return errors.New("bisect details are required")
		}
	case KindUnknown:
		if details.Rebase != nil || details.CherryPick != nil || details.Revert != nil || details.Merge != nil || details.Bisect != nil {
			return errors.New("unknown operation cannot carry typed details")
		}
	}
	return nil
}

func cloneDetails(details Details) Details {
	copyDetails := details
	if details.Rebase != nil {
		value := *details.Rebase
		copyDetails.Rebase = &value
	}
	if details.CherryPick != nil {
		value := *details.CherryPick
		value.Commits = append([]string(nil), value.Commits...)
		copyDetails.CherryPick = &value
	}
	if details.Revert != nil {
		value := *details.Revert
		value.Commits = append([]string(nil), value.Commits...)
		copyDetails.Revert = &value
	}
	if details.Merge != nil {
		value := *details.Merge
		copyDetails.Merge = &value
	}
	if details.Bisect != nil {
		value := *details.Bisect
		copyDetails.Bisect = &value
	}
	return copyDetails
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func maxZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
