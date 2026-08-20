package commitview

import (
	"fmt"

	"github.com/jusanchez/gitwatch/internal/commitmodel"
	"strings"
)

type Composer struct {
	Draft         commitmodel.Draft
	Focus         string
	Width, Height int
	Error         string
	ConfigSummary string
}

func New(files []commitmodel.File) Composer {
	return Composer{Draft: commitmodel.Draft{Files: append([]commitmodel.File(nil), files...)}, Focus: "subject"}
}
func (c *Composer) SetSubject(s string)             { c.Draft.Subject = s; c.Error = "" }
func (c *Composer) SetBody(s string)                { c.Draft.Body = s; c.Error = "" }
func (c *Composer) SetConfigSummary(summary string) { c.ConfigSummary = summary }
func (c Composer) Ready() bool                      { return c.Draft.Validate().Valid }
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
