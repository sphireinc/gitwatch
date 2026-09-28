package customcmd

import (
	"strings"
	"testing"
)

func TestFormLengthValidationKeepsValuesPrivateUntilSubmit(t *testing.T) {
	for _, kind := range []PromptKind{PromptText, PromptSecret} {
		t.Run(string(kind), func(t *testing.T) {
			form, err := NewForm([]Prompt{{ID: "value", Label: "Value", Kind: kind, MinLength: 2, MaxLength: 3}})
			if err != nil {
				t.Fatal(err)
			}
			if event, err := form.Handle("enter"); event != FormChanged || err == nil {
				t.Fatalf("empty submit: %v, %v", event, err)
			}
			for _, key := range []string{"é", "界", "🙂", "x"} {
				if _, err := form.Handle(key); err != nil {
					t.Fatal(err)
				}
			}
			if event, err := form.Handle("enter"); event != FormChanged || err == nil || strings.Contains(err.Error(), "é界🙂x") {
				t.Fatalf("overlength submit: %v, %v", event, err)
			}
			if values := form.Values(); len(values) != 0 {
				t.Fatalf("incomplete form exposed values: %v", values)
			}
			if _, err := form.Handle("backspace"); err != nil {
				t.Fatal(err)
			}
			if event, err := form.Handle("enter"); event != FormSubmitted || err != nil {
				t.Fatalf("valid unicode submit: %v, %v", event, err)
			}
			if got := form.Values()["value"]; got != "é界🙂" {
				t.Fatalf("unicode value = %q", got)
			}
		})
	}
}

func TestFormRejectsInvalidLengthDefinitions(t *testing.T) {
	for _, prompt := range []Prompt{
		{Kind: PromptText, MinLength: -1},
		{Kind: PromptText, MaxLength: -1},
		{Kind: PromptText, MinLength: 3, MaxLength: 2},
		{Kind: PromptText, MinLength: 2, Default: "é"},
		{Kind: PromptText, MaxLength: 1, Default: "é界"},
		{Kind: PromptConfirm, MaxLength: 3},
		{Kind: PromptSelect, Options: []string{"one"}, MinLength: 1},
	} {
		prompt.ID, prompt.Label = "value", "Value"
		if _, err := NewForm([]Prompt{prompt}); err == nil {
			t.Errorf("accepted invalid length definition: %+v", prompt)
		}
	}
}

func TestSelectionValidationReportsErrorAndAllowsCorrection(t *testing.T) {
	for _, kind := range []PromptKind{PromptSelect, PromptMultiSelect} {
		t.Run(string(kind), func(t *testing.T) {
			form, err := NewForm([]Prompt{{ID: "choice", Label: "Choice", Kind: kind, Required: true, Pattern: "^valid$", Options: []string{"invalid", "valid"}}})
			if err != nil {
				t.Fatal(err)
			}
			if kind == PromptMultiSelect {
				if event, err := form.Handle("enter"); event != FormChanged || err == nil {
					t.Fatalf("required submit: %v, %v", event, err)
				}
				if _, err := form.Handle("space"); err != nil {
					t.Fatal(err)
				}
			}
			if event, err := form.Handle("enter"); event != FormChanged || err == nil {
				t.Fatalf("pattern submit: %v, %v", event, err)
			}
			if values := form.Values(); len(values) != 0 {
				t.Fatalf("invalid selection exposed values: %v", values)
			}
			if kind == PromptMultiSelect {
				if _, err := form.Handle("space"); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := form.Handle("down"); err != nil {
				t.Fatal(err)
			}
			if kind == PromptMultiSelect {
				if _, err := form.Handle("space"); err != nil {
					t.Fatal(err)
				}
			}
			if event, err := form.Handle("enter"); event != FormSubmitted || err != nil {
				t.Fatalf("corrected submit: %v, %v", event, err)
			}
		})
	}
}

func TestPromptAndPathValuesAreExpandedExactlyOnce(t *testing.T) {
	definition := Definition{
		Name: "literal-values", Executable: "tool",
		Args:    []string{"{prompt:value}", "prefix={path}/{branch}", "{prompt:value}{prompt:value}"},
		Prompts: []Prompt{{ID: "value", Label: "Value", Kind: PromptText}},
	}
	ctx := Context{Branch: "main", SelectedPath: "dir/{branch}/{prompt:value}", PromptValues: map[string]string{"value": "{path} $() 'quoted'"}}
	for range 100 {
		invocation, err := definition.Expand(ctx)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"{path} $() 'quoted'", "prefix=dir/{branch}/{prompt:value}/main", "{path} $() 'quoted'{path} $() 'quoted'"}
		for index, value := range want {
			if invocation.Args[index] != value {
				t.Fatalf("arg %d = %q, want %q", index, invocation.Args[index], value)
			}
		}
	}
}
