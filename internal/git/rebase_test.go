package git

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sphireinc/git-watch/internal/rebase"
)

func TestHandleSequenceEditorAtomicallyInstallsPrivatePlan(t *testing.T) {
	root := t.TempDir()
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "rebase-merge"), 0o700); err != nil {
		t.Fatal(err)
	}
	todoPath := filepath.Join(gitDir, "rebase-merge", "git-rebase-todo")
	if err := os.WriteFile(todoPath, []byte("pick old old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := []byte("pick abc123 safe subject\n# retained\n")
	planPath := filepath.Join(root, "plan.todo")
	if err := os.WriteFile(planPath, plan, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(plan)
	manifest := sequenceManifest{OperationID: "op_123", Repository: root, GitDir: gitDir, PlanPath: planPath, PlanSHA256: hex.EncodeToString(digest[:])}
	manifestPath := filepath.Join(root, "manifest.json")
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := HandleSequenceEditor([]string{todoPath}, []string{sequenceManifestEnv + "=" + manifestPath, sequenceOperationEnv + "=op_123"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(todoPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(plan) {
		t.Fatalf("todo = %q", got)
	}
}

func TestHandleSequenceEditorRejectsUntrustedInvocation(t *testing.T) {
	if err := HandleSequenceEditor(nil, nil); err == nil {
		t.Fatal("missing arguments were accepted")
	}
	if err := HandleSequenceEditor([]string{"/tmp/git-rebase-todo"}, []string{sequenceOperationEnv + "=bad/path"}); err == nil {
		t.Fatal("unscoped operation was accepted")
	}
}

func TestStartInteractiveRebaseRejectsInvalidRequestsBeforeGit(t *testing.T) {
	runner := NewRunner(t.TempDir())
	plan, err := rebase.Parse("squash abc invalid\npick def second\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.StartInteractiveRebase(t.Context(), RebaseRequest{OperationID: "safe", Base: "HEAD~1", Plan: plan}); err == nil || !strings.Contains(err.Error(), "first commit") {
		t.Fatalf("invalid plan error = %v", err)
	}
	valid, err := rebase.Parse("pick abc first\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.StartInteractiveRebase(t.Context(), RebaseRequest{OperationID: "bad/id", Base: "HEAD~1", Plan: valid}); err == nil || !strings.Contains(err.Error(), "operation ID") {
		t.Fatalf("invalid operation ID error = %v", err)
	}
}
