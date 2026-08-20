package branches

import "strings"

// ParseTracking maps %(upstream:trackshort) symbols to divergence counts. A
// count is represented as one when Git only supplies a symbol.
func ParseTracking(value string) (ahead, behind int) {
	switch strings.TrimSpace(value) {
	case ">":
		return 1, 0
	case "<":
		return 0, 1
	case "<>":
		return 1, 1
	case "=":
		return 0, 0
	default:
		return 0, 0
	}
}
