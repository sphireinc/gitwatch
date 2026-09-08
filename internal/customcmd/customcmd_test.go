package customcmd

import (
	"context"
	"os"
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
	if !buffer.exceeded || string(buffer.Bytes()) != "123" {
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
