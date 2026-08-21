package branches

import (
	"fmt"
	"strconv"
	"strings"
)

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

func ParseDivergence(value []byte) (behind, ahead int, err error) {
	fields := strings.Fields(string(value))
	if len(fields) != 2 {
		return 0, 0, fmt.Errorf("invalid divergence count %q", strings.TrimSpace(string(value)))
	}
	behind, err = strconv.Atoi(fields[0])
	if err != nil || behind < 0 {
		return 0, 0, fmt.Errorf("invalid upstream divergence %q", fields[0])
	}
	ahead, err = strconv.Atoi(fields[1])
	if err != nil || ahead < 0 {
		return 0, 0, fmt.Errorf("invalid local divergence %q", fields[1])
	}
	return behind, ahead, nil
}
