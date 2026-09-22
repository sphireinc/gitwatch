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
