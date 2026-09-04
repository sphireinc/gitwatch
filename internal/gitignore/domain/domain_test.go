package domain

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestTemplateIDParsingAndRoundTrip(t *testing.T) {
	cases := []struct {
		value string
		valid bool
	}{
		{"root/PHP", true}, {"global/macOS", true}, {"community/Java/Gradle", true},
		{"root/../PHP", false}, {"root//PHP", false}, {"other/PHP", false}, {"root\\PHP", false}, {"PHP", false},
	}
	for _, tc := range cases {
		id, err := ParseTemplateID(tc.value)
		if (err == nil) != tc.valid {
			t.Errorf("ParseTemplateID(%q) error=%v, valid=%v", tc.value, err, tc.valid)
		}
		if tc.valid && id.String() != tc.value {
			t.Errorf("ID changed: %q", id)
		}
	}
	state := struct {
		ID TemplateID `json:"id"`
	}{ID: "community/Java/Gradle"}
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		ID TemplateID `json:"id"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != state.ID {
		t.Fatalf("round trip = %q, want %q", decoded.ID, state.ID)
	}
}

func TestSnapshotAndMutationPlanOwnMutableData(t *testing.T) {
	snapshot, err := NewDocumentSnapshot("repo-a", "/work/repo-a", ".gitignore", []byte("*.log\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Newline != NewlineLF || snapshot.SHA256 == "" || !snapshot.FinalNewline {
		t.Fatalf("bad snapshot metadata: %+v", snapshot)
	}
	content := []byte("new\n")
	plan, err := NewMutationPlan(snapshot, MutationAppend, []TemplateID{"root/PHP"}, []Edit{{Replacement: content}}, content, []string{"preview"})
	if err != nil {
		t.Fatal(err)
	}
	content[0] = 'x'
	snapshot.Bytes[0] = 'x'
	if string(plan.ResultBytes) != "new\n" || string(plan.BeforeBytes) != "*.log\n" {
		t.Fatal("mutation plan did not copy preview data")
	}
}

func TestMatchOwnershipIsExplicit(t *testing.T) {
	block := &ManagedBlock{TemplateID: "root/PHP", Start: 0, End: 3, ContentSHA256: "hash"}
	if !(TemplateMatch{Kind: ManagedExact, Block: block}).Owned() {
		t.Fatal("valid managed block should be owned")
	}
	if (TemplateMatch{Kind: UnmanagedFull}).Owned() {
		t.Fatal("unmanaged full match must not be owned")
	}
	for _, kind := range []MatchKind{Partial, Absent, InvalidManagedBlock} {
		if (TemplateMatch{Kind: kind, Block: block}).Owned() {
			t.Errorf("%s unexpectedly owned", kind)
		}
	}
}

func TestUnsafeSnapshot(t *testing.T) {
	_, err := NewDocumentSnapshot("repo-a", "/work/repo-a", "../.gitignore", nil, 0)
	if !errors.Is(err, ErrUnsafeTarget) {
		t.Fatalf("error=%v, want ErrUnsafeTarget", err)
	}
}
