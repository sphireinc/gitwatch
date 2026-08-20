package app

import "fmt"

type PartialDiscard struct {
	Path                 string
	HunkCount, LineCount int
	Confirmed            bool
}

func (d PartialDiscard) Prompt() string {
	return fmt.Sprintf("Discard %d selected line(s) across %d hunk(s) in %s? Type 'discard' to confirm.", d.LineCount, d.HunkCount, d.Path)
}
func (d PartialDiscard) Accept(input string) bool { return d.Confirmed && input == "discard" }
