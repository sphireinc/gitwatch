package git

import (
	"context"
	"fmt"
)

// ShowPath reads one exact path from an immutable commit object with a byte
// bound. The commit ID must be a resolved full object ID, so user input cannot
// become a revision expression or an option.
func (r Runner) ShowPath(ctx context.Context, commit string, path []byte, maxBytes int) ([]byte, error) {
	if !validObjectID(commit) {
		return nil, fmt.Errorf("resolved commit ID is required")
	}
	if len(path) == 0 {
		return nil, fmt.Errorf("historical path is required")
	}
	if maxBytes <= 0 {
		return nil, fmt.Errorf("historical file size limit must be positive")
	}
	result, err := r.RunBounded(ctx, maxBytes, "show", "--format=", "--no-ext-diff", commit+":"+string(path))
	return result.Stdout, err
}
