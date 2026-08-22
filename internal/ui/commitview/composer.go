// Package commitview renders and validates the commit composer.
package commitview

import (
	"fmt"

	"github.com/sphireinc/git-watch/internal/commitmodel"
	"strings"
)

// Composer stores editable commit draft state and validation feedback.
type Composer struct {
	Draft         commitmodel.Draft
	Focus         string
	Width, Height int
	Error         string
	ConfigSummary string
}

// New creates a composer for the supplied commit files.
func New(files []commitmodel.File) Composer {
	return Composer{Draft: commitmodel.Draft{Files: append([]commitmodel.File(nil), files...)}, Focus: "subject"}
}

// SetSubject updates the commit subject and clears the prior error.
func (c *Composer) SetSubject(s string) { c.Draft.Subject = s; c.Error = "" }

// SetBody updates the commit body and clears the prior error.
func (c *Composer) SetBody(s string) { c.Draft.Body = s; c.Error = "" }

// SetConfigSummary updates the displayed commit configuration summary.
func (c *Composer) SetConfigSummary(summary string) { c.ConfigSummary = summary }

// Ready reports whether the current draft passes validation.
func (c Composer) Ready() bool { return c.Draft.Validate().Valid }

// View renders the commit composer and validation messages.
func (c Composer) View() string {
	v := c.Draft.Validate()
	lines := []string{"Commit changes", "", "Staged files:"}
	for _, f := range c.Draft.Files {
		if f.Staged {
			lines = append(lines, "  ✓ "+f.Path)
		}
	}
	subjectMarker, bodyMarker := "  ", "  "
	if c.Focus == "subject" {
		subjectMarker = "> "
	} else {
		bodyMarker = "> "
	}
	lines = append(lines, "", subjectMarker+"Subject: "+c.Draft.Subject, "", bodyMarker+c.Draft.Body, "", fmt.Sprintf("Options: [A] amend=%t [N] no-edit=%t [o] signoff=%t [S] sign=%t [@] author=%q", c.Draft.Amend, c.Draft.NoEdit, c.Draft.Signoff, c.Draft.Sign, c.Draft.Author))
	if c.ConfigSummary != "" {
		lines = append(lines, "  "+c.ConfigSummary)
	}
	lines = append(lines, "────────────────────────────────────────")
	if len(v.Errors) > 0 {
		lines = append(lines, "Error: "+strings.Join(v.Errors, "; "))
	} else if len(v.Warnings) > 0 {
		lines = append(lines, "Warning: "+strings.Join(v.Warnings, "; "))
	} else {
		lines = append(lines, "Ready to commit")
	}
	return strings.Join(lines, "\n")
}
