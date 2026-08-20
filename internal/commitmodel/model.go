package commitmodel

import (
	"fmt"
	"strings"
)

type File struct {
	Path          string
	Staged        bool
	SelectedHunks int
}
type Draft struct {
	Subject, Body                string
	Amend, NoEdit, Signoff, Sign bool
	Author                       string
	Files                        []File
}
type Validation struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

func (d Draft) Message() string {
	if d.NoEdit {
		return ""
	}
	if d.Body == "" {
		return strings.TrimSpace(d.Subject) + "\n"
	}
	return strings.TrimSpace(d.Subject) + "\n\n" + strings.TrimSpace(d.Body) + "\n"
}
func (d Draft) Validate() Validation {
	v := Validation{Valid: true}
	if d.Amend && d.NoEdit && d.Subject != "" {
		v.Warnings = append(v.Warnings, "no-edit ignores the draft message")
	}
	if !d.NoEdit && strings.TrimSpace(d.Subject) == "" {
		v.Valid = false
		v.Errors = append(v.Errors, "commit subject is required")
	}
	if len(d.Subject) > 72 {
		v.Warnings = append(v.Warnings, fmt.Sprintf("subject is %d characters; keep it under 72 when possible", len(d.Subject)))
	}
	if strings.ContainsAny(d.Author, "\r\n") {
		v.Valid = false
		v.Errors = append(v.Errors, "author must be one line")
	}
	staged := 0
	for _, f := range d.Files {
		if f.Staged {
			staged++
		}
	}
	if staged == 0 && !d.Amend {
		v.Valid = false
		v.Errors = append(v.Errors, "at least one staged path is required")
	}
	return v
}
func (d Draft) WithSubject(subject string) Draft { d.Subject = subject; return d }
func (d Draft) ClearAfterSuccess() Draft         { return Draft{} }
