package branches

import "fmt"

type Confirmation struct {
	Name  string
	Force bool
}

func DeletePrompt(name string, force bool) Confirmation {
	return Confirmation{Name: name, Force: force}
}
func (c Confirmation) Text() string {
	if c.Force {
		return fmt.Sprintf("Force-delete branch %s? This may discard commits.", c.Name)
	}
	return fmt.Sprintf("Delete branch %s?", c.Name)
}
func (c Confirmation) Accept(input string) bool { return input == c.Name }
