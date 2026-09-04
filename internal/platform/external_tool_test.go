package platform

import "testing"

func TestExternalToolCommandKeepsPathAsOneArg(t *testing.T) {
	command, err := (ExternalTool{Executable: "merge-tool", Args: []string{"--file={path}", "--mode", "interactive"}}).Command("space name/file.txt", "/repo", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(command.Args) != 4 || command.Args[1] != "--file=space name/file.txt" || command.Args[2] != "--mode" {
		t.Fatalf("command args = %#v", command.Args)
	}
}

func TestExternalToolRejectsMissingConfiguration(t *testing.T) {
	if _, err := (ExternalTool{}).Command("file", "/repo", nil); err == nil {
		t.Fatal("expected missing executable error")
	}
	if _, err := (ExternalTool{Executable: "tool"}).Command("", "/repo", nil); err == nil {
		t.Fatal("expected missing path error")
	}
}
