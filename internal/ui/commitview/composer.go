package commitview

import (
	"github.com/jusanchez/gitwatch/internal/commitmodel"
	"strings"
)

type Composer struct {
	Draft         commitmodel.Draft
	Focus         string
	Width, Height int
	Error         string
}

func New(files []commitmodel.File) Composer {
	return Composer{Draft: commitmodel.Draft{Files: append([]commitmodel.File(nil), files...)}, Focus: "subject"}
}
func (c *Composer) SetSubject(s string) { c.Draft.Subject = s; c.Error = "" }
func (c *Composer) SetBody(s string)    { c.Draft.Body = s; c.Error = "" }
func (c Composer) Ready() bool          { return c.Draft.Validate().Valid }
func (c Composer) View() string {
	v := c.Draft.Validate()
	lines := []string{"Commit changes", "", "Staged files:"}
	for _, f := range c.Draft.Files {
		if f.Staged {
			lines = append(lines, "  ✓ "+f.Path)
		}
	}
	lines = append(lines, "", "Subject: "+c.Draft.Subject, "", c.Draft.Body, "", "────────────────────────────────────────")
	if len(v.Errors) > 0 {
		lines = append(lines, "Error: "+strings.Join(v.Errors, "; "))
	} else if len(v.Warnings) > 0 {
		lines = append(lines, "Warning: "+strings.Join(v.Warnings, "; "))
	} else {
		lines = append(lines, "Ready to commit")
	}
	return strings.Join(lines, "\n")
}
