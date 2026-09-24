package customcmd

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestExpandKeepsInjectedValuesInOneArg(t *testing.T) {
	definition := Definition{Name: "inspect", Executable: "tool", Args: []string{"--path={path}", "{branch}"}, Mutates: true}
	invocation, err := definition.Expand(Context{SelectedPath: "space name; echo unsafe", Branch: "feature/ref"})
	if err != nil {
		t.Fatal(err)
	}
	if len(invocation.Args) != 2 || invocation.Args[0] != "--path=space name; echo unsafe" || invocation.Args[1] != "feature/ref" {
		t.Fatalf("args = %#v", invocation.Args)
	}
	if !invocation.Refresh {
		t.Fatal("mutating command did not force refresh")
	}
	command, err := invocation.Command()
	if err != nil || command.Args[1] != invocation.Args[0] {
		t.Fatalf("command=%#v err=%v", command, err)
	}
}

func TestExpandRejectsMissingAndUnknownContext(t *testing.T) {
	definition := Definition{Name: "open", Executable: "tool", Args: []string{"{path}"}}
	if _, err := definition.Expand(Context{}); err == nil {
		t.Fatal("missing path context was accepted")
	}
	definition.Args = []string{"{unknown}"}
	if _, err := definition.Expand(Context{}); err == nil {
		t.Fatal("unknown placeholder was accepted")
	}
}

func TestValidateRejectsShellControlInArg(t *testing.T) {
	definition := Definition{Name: "unsafe", Executable: "tool", Args: []string{"a\n b"}}
	if _, err := definition.Expand(Context{}); err == nil {
		t.Fatal("newline argv value was accepted")
	}
}

func TestRunBoundsOutputAndHonorsCancellation(t *testing.T) {
	buffer := &limitedBuffer{limit: 3}
	_, _ = buffer.Write([]byte("123456"))
	if !buffer.exceeded || buffer.String() != "123" {
		t.Fatalf("limited buffer = %q exceeded=%v", buffer.Bytes(), buffer.exceeded)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	invocation := Invocation{Name: "cancel", Executable: os.Args[0], Args: []string{"-test.run=TestCustomCmdHelper"}}
	_, err := Run(ctx, invocation, 100)
	if err == nil {
		t.Fatalf("cancellation error = %v", err)
	}
}

func TestFormValidatesTextSelectSecretAndCancelWithoutSubmitting(t *testing.T) {
	form, err := NewForm([]Prompt{
		{ID: "ticket", Label: "Ticket", Kind: PromptText, Required: true, Pattern: `^[A-Z]+-[0-9]+$`},
		{ID: "branch", Label: "Branch", Kind: PromptSelect, Options: []string{"main", "feature"}},
		{ID: "token", Label: "Token", Kind: PromptSecret, Required: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event, err := form.Handle("X"); err != nil || event != FormChanged {
		t.Fatalf("text event = %v, err=%v", event, err)
	}
	if event, err := form.Handle("enter"); err == nil || event != FormChanged {
		t.Fatalf("invalid text event = %v, err=%v", event, err)
	}
	if _, err := form.Handle("backspace"); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"A", "-", "1", "2", "3"} {
		if _, err := form.Handle(key); err != nil {
			t.Fatal(err)
		}
	}
	if event, err := form.Handle("enter"); err != nil || event != FormChanged {
		t.Fatalf("accepted text event = %v, err=%v", event, err)
	}
	if _, err := form.Handle("down"); err != nil {
		t.Fatal(err)
	}
	if event, err := form.Handle("enter"); err != nil || event != FormChanged {
		t.Fatalf("select event = %v, err=%v", event, err)
	}
	if _, err := form.Handle("s"); err != nil {
		t.Fatal(err)
	}
	if event, err := form.Handle("esc"); err != nil || event != FormCancelled {
		t.Fatalf("cancel event = %v, err=%v", event, err)
	}
	if values := form.Values(); values != nil {
		t.Fatalf("cancelled form exposed values: %#v", values)
	}
}

func TestFormSubmitsTypedValuesAndRedactsSecrets(t *testing.T) {
	form, err := NewForm([]Prompt{
		{ID: "confirm", Label: "Confirm", Kind: PromptConfirm},
		{ID: "targets", Label: "Targets", Kind: PromptMultiSelect, Options: []string{"one", "two"}},
		{ID: "token", Label: "Token", Kind: PromptSecret},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event, err := form.Handle("y"); err != nil || event != FormChanged {
		t.Fatalf("confirm event = %v, err=%v", event, err)
	}
	if _, err := form.Handle("space"); err != nil {
		t.Fatal(err)
	}
	if _, err := form.Handle("down"); err != nil {
		t.Fatal(err)
	}
	if _, err := form.Handle("space"); err != nil {
		t.Fatal(err)
	}
	if event, err := form.Handle("enter"); err != nil || event != FormChanged {
		t.Fatalf("multi-select event = %v, err=%v", event, err)
	}
	for _, key := range []string{"s", "e", "c", "r", "e", "t"} {
		if _, err := form.Handle(key); err != nil {
			t.Fatal(err)
		}
	}
	event, err := form.Handle("enter")
	if err != nil || event != FormSubmitted {
		t.Fatalf("submit event = %v, err=%v", event, err)
	}
	values := form.Values()
	if values["confirm"] != "true" || values["targets"] != "one,two" || values["token"] != "secret" {
		t.Fatalf("submitted values = %#v", values)
	}
	if redacted := form.RedactedValues(); redacted["token"] != "[redacted]" || redacted["targets"] != "one,two" {
		t.Fatalf("redacted values = %#v", redacted)
	}
}

func TestDefinitionPromptsExpandOnlyAfterValuesAreSupplied(t *testing.T) {
	definition := Definition{Name: "ticket", Executable: "tool", Prompts: []Prompt{{ID: "ticket", Label: "Ticket", Kind: PromptText, Required: true}}, Args: []string{"--ticket={prompt:ticket}"}}
	if _, err := definition.Expand(Context{}); err == nil {
		t.Fatal("missing prompt value was accepted")
	}
	invocation, err := definition.Expand(Context{PromptValues: map[string]string{"ticket": "ABC-42"}})
	if err != nil || len(invocation.Args) != 1 || invocation.Args[0] != "--ticket=ABC-42" {
		t.Fatalf("prompt expansion = %#v, err=%v", invocation, err)
	}
}

func TestResolvePromptsUsesLoadedRepositoryOptionsWithoutRunningCommands(t *testing.T) {
	prompts := []Prompt{{ID: "branch", Label: "Branch", Kind: PromptSelect, OptionsSource: "branches"}}
	resolved, err := ResolvePrompts(prompts, Context{OptionValues: map[string][]string{"branches": {"main", "feature"}}})
	if err != nil || len(resolved) != 1 || len(resolved[0].Options) != 2 || resolved[0].Options[1] != "feature" {
		t.Fatalf("resolved prompts = %#v, err=%v", resolved, err)
	}
	if _, err := ResolvePrompts(prompts, Context{}); err == nil {
		t.Fatal("empty dynamic option source was accepted")
	}
}

func TestRunSuppressesSecretPromptOutputAndErrorDetails(t *testing.T) {
	const secret = "private-token-value"
	for _, mode := range []string{"stdout", "stderr"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("GITWATCH_CUSTOMCMD_HELPER_SECRET", secret)
			t.Setenv("GITWATCH_CUSTOMCMD_HELPER_MODE", mode)
			definition := Definition{
				Name:       "secret-test",
				Executable: os.Args[0],
				Args:       []string{"-test.run=^TestCustomCmdHelper$"},
				Prompts:    []Prompt{{ID: "token", Label: "Token", Kind: PromptSecret}},
			}
			invocation, err := definition.Expand(Context{PromptValues: map[string]string{"token": secret}})
			if err != nil {
				t.Fatal(err)
			}
			output, runErr := Run(context.Background(), invocation, 1024)
			if !output.Suppressed || len(output.Stdout) != 0 || len(output.Stderr) != 0 {
				t.Fatalf("secret output escaped: %#v", output)
			}
			if mode == "stderr" {
				if runErr == nil || strings.Contains(runErr.Error(), secret) {
					t.Fatalf("secret-bearing command error = %v", runErr)
				}
			} else if runErr != nil {
				t.Fatalf("successful secret command failed: %v", runErr)
			}
		})
	}
}

func TestCustomCmdHelper(t *testing.T) {
	secret := os.Getenv("GITWATCH_CUSTOMCMD_HELPER_SECRET")
	if secret == "" {
		return
	}
	if os.Getenv("GITWATCH_CUSTOMCMD_HELPER_MODE") == "stderr" {
		t.Fatalf("helper emitted secret: %s", secret)
	}
	if _, err := os.Stdout.WriteString(secret); err != nil {
		t.Fatal(err)
	}
}
