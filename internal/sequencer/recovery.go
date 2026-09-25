package sequencer

// RecoveryView identifies the host workspace that presents a durable Git
// operation. It is deliberately UI-neutral; callers map it to their own
// workspace enum.
type RecoveryView uint8

const (
	RecoveryConflictView RecoveryView = iota
	RecoveryCherryPickView
	RecoveryBisectView
)

// RecoveryRoute is the single operation-kind-to-recovery presentation map.
// Lifecycle execution remains at the Git boundary; this type only describes
// where a freshly observed operation can be reopened.
type RecoveryRoute struct {
	View  RecoveryView
	Label string
}

// RecoveryActions describes the lifecycle actions that are valid for a
// Git-derived operation projection. It is UI-neutral so every workspace uses
// the same safety decision before rendering or dispatching an action.
type RecoveryActions struct {
	Continue bool
	Abort    bool
	Skip     bool
}

// ActionsFor derives valid lifecycle actions from the operation kind and
// authoritative conflict/index projection. A negative staged count means the
// caller has not supplied an index count and the coordinator remains
// conservative only where the count is known.
func ActionsFor(state *State, unresolved, staged int) RecoveryActions {
	if state == nil {
		return RecoveryActions{}
	}
	kind := state.Kind()
	if kind == KindUnknown {
		return RecoveryActions{}
	}
	actions := RecoveryActions{Abort: true}
	switch kind {
	case KindRebase, KindCherryPick, KindRevert:
		actions.Skip = true
	case KindMerge:
	default:
		return RecoveryActions{}
	}
	if unresolved > 0 {
		return actions
	}
	if kind == KindRebase {
		details := state.Details().Rebase
		if details != nil && details.EditStopped {
			actions.Continue = true
		} else if staged > 0 {
			actions.Continue = true
		}
		return actions
	}
	// Cherry-pick, revert, and merge require an authoritative staged result
	// before Continue is offered. A missing/unknown count is conservative.
	actions.Continue = staged > 0
	return actions
}

// RouteFor returns the common recovery route for every supported durable Git
// operation. Unknown operations are intentionally not routed.
func RouteFor(kind Kind) (RecoveryRoute, bool) {
	switch kind {
	case KindCherryPick:
		return RecoveryRoute{View: RecoveryCherryPickView, Label: "Cherry-pick progress"}, true
	case KindBisect:
		return RecoveryRoute{View: RecoveryBisectView, Label: "Bisect"}, true
	case KindRebase:
		return RecoveryRoute{View: RecoveryConflictView, Label: "Rebase recovery"}, true
	case KindRevert:
		return RecoveryRoute{View: RecoveryConflictView, Label: "Revert recovery"}, true
	case KindMerge:
		return RecoveryRoute{View: RecoveryConflictView, Label: "Merge recovery"}, true
	default:
		return RecoveryRoute{}, false
	}
}
