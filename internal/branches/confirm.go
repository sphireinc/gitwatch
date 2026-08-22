package branches

import "fmt"

// Confirmation describes the exact text required before a branch mutation.
type Confirmation struct {
	Name  string
	Force bool
}

// DeletePrompt creates a confirmation for deleting name.
func DeletePrompt(name string, force bool) Confirmation {
	return Confirmation{Name: name, Force: force}
}

// Text returns the user-facing confirmation prompt.
func (c Confirmation) Text() string {
	if c.Force {
		return fmt.Sprintf("Force-delete branch %s? This may discard commits.", c.Name)
	}
	return fmt.Sprintf("Delete branch %s?", c.Name)
}

// Accept reports whether input exactly confirms the requested branch.
func (c Confirmation) Accept(input string) bool { return input == c.Name }
