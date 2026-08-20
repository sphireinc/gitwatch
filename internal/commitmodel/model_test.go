package commitmodel

import "testing"

func TestDraftValidationAndPreservation(t *testing.T) {
	d := Draft{Subject: "add feature", Files: []File{{Path: "a", Staged: true}}}
	if !d.Validate().Valid || !d.Validate().Valid {
		t.Fatal(d.Validate())
	}
	if d.Message() != "add feature\n" {
		t.Fatal(d.Message())
	}
	if d.ClearAfterSuccess().Subject != "" {
		t.Fatal("draft not cleared")
	}
	d.Subject = ""
	if d.Validate().Valid {
		t.Fatal("empty subject accepted")
	}
}
