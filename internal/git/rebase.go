package git

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sphireinc/git-watch/internal/rebase"
	"github.com/sphireinc/git-watch/internal/sequencer"
)

const (
	sequenceManifestEnv  = "GITWATCH_REBASE_MANIFEST"
	sequenceOperationEnv = "GITWATCH_REBASE_OPERATION"
	sequenceEditorEnv    = "GIT_SEQUENCE_EDITOR"
)

// RebaseRequest describes one validated interactive rebase invocation.
type RebaseRequest struct {
	OperationID string
	Base        string
	Root        bool
	Autosquash  bool
	Plan        rebase.Plan
	Editor      string
}

// RebaseOutcome retains the Git result and the authoritative operation probe.
// A paused rebase is reported through Paused/State, even when Git exits
// non-zero because it stopped for a conflict or another recovery action.
type RebaseOutcome struct {
	OperationID string
	Result      Result
	CommandErr  error
	Paused      bool
	State       *sequencer.State
}

type sequenceManifest struct {
	OperationID string `json:"operation_id"`
	Repository  string `json:"repository"`
	GitDir      string `json:"git_dir"`
	PlanPath    string `json:"plan_path"`
	PlanSHA256  string `json:"plan_sha256"`
}

// StartInteractiveRebase validates and starts an interactive rebase without
// opening an external todo editor. Git remains the source of truth for the
// resulting operation state.
func (r Runner) StartInteractiveRebase(ctx context.Context, request RebaseRequest) (RebaseOutcome, error) {
	if err := request.Plan.Validate(); err != nil {
		return RebaseOutcome{}, fmt.Errorf("rebase plan: %w", err)
	}
	if !request.Root && strings.TrimSpace(request.Base) == "" {
		return RebaseOutcome{}, errors.New("rebase base is required unless root mode is selected")
	}
	if request.Root && request.Base != "" {
		return RebaseOutcome{}, errors.New("rebase base cannot be combined with root mode")
	}
	operationID := request.OperationID
	if operationID == "" {
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return RebaseOutcome{}, fmt.Errorf("generate rebase operation ID: %w", err)
		}
		operationID = hex.EncodeToString(bytes[:])
	}
	if !validOperationID(operationID) {
		return RebaseOutcome{}, errors.New("rebase operation ID must contain only letters, digits, hyphens, or underscores")
	}

	discovery, err := Discover(ctx, r.Dir)
	if err != nil {
		return RebaseOutcome{}, err
	}
	workspace, err := os.MkdirTemp("", ".gitwatch-rebase-"+operationID+"-")
	if err != nil {
		return RebaseOutcome{}, fmt.Errorf("create rebase handoff: %w", err)
	}
	defer os.RemoveAll(workspace)
	if err := os.Chmod(workspace, 0o700); err != nil {
		return RebaseOutcome{}, fmt.Errorf("protect rebase handoff: %w", err)
	}
	planPath := filepath.Join(workspace, "plan.todo")
	planBytes := []byte(request.Plan.Render())
	if err := os.WriteFile(planPath, planBytes, 0o600); err != nil {
		return RebaseOutcome{}, fmt.Errorf("write rebase plan: %w", err)
	}
	digest := sha256.Sum256(planBytes)
	manifest := sequenceManifest{OperationID: operationID, Repository: discovery.Root, GitDir: discovery.GitDir, PlanPath: planPath, PlanSHA256: hex.EncodeToString(digest[:])}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return RebaseOutcome{}, fmt.Errorf("encode rebase handoff: %w", err)
	}
	manifestPath := filepath.Join(workspace, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestBytes, 0o600); err != nil {
		return RebaseOutcome{}, fmt.Errorf("write rebase handoff: %w", err)
	}

	editor := request.Editor
	if editor == "" {
		editor = "gitwatch internal sequence-editor"
	}
	env := append([]string{}, r.Env...)
	env = append(env, sequenceEditorEnv+"="+editor, sequenceManifestEnv+"="+manifestPath, sequenceOperationEnv+"="+operationID)
	runner := r
	runner.Env = env
	args := []string{"rebase", "-i"}
	if request.Autosquash {
		args = append(args, "--autosquash")
	}
	if request.Root {
		args = append(args, "--root")
	} else {
		args = append(args, request.Base)
	}
	result, commandErr := runner.Run(ctx, args...)
	outcome := RebaseOutcome{OperationID: operationID, Result: result, CommandErr: commandErr}
	updated, probeErr := Discover(ctx, r.Dir)
	if probeErr == nil {
		operation, detectErr := DetectOperationState(ctx, updated, 0)
		if detectErr == nil && operation.Found {
			outcome.Paused = true
			state := operation.State
			outcome.State = &state
		}
	}
	return outcome, commandErr
}

func validOperationID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '-' && character != '_' {
			return false
		}
	}
	return true
}

// HandleSequenceEditor is the private helper boundary used by Git through
// GIT_SEQUENCE_EDITOR. It accepts only a todo path and a matching active
// handoff manifest; arbitrary invocations are rejected.
func HandleSequenceEditor(args []string, environment []string) error {
	if len(args) != 1 {
		return errors.New("sequence-editor requires exactly one todo path")
	}
	values := make(map[string]string, 3)
	for _, item := range environment {
		key, value, ok := strings.Cut(item, "=")
		if ok && (key == sequenceManifestEnv || key == sequenceOperationEnv) {
			values[key] = value
		}
	}
	manifestPath, operationID := values[sequenceManifestEnv], values[sequenceOperationEnv]
	if manifestPath == "" || operationID == "" || !validOperationID(operationID) {
		return errors.New("sequence-editor handoff is not active")
	}
	manifestInfo, err := os.Stat(manifestPath)
	if err != nil || manifestInfo.Mode().Perm()&0o077 != 0 {
		return errors.New("sequence-editor handoff is inaccessible or not private")
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read sequence-editor handoff: %w", err)
	}
	var manifest sequenceManifest
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.OperationID != operationID || !validOperationID(manifest.OperationID) {
		return errors.New("sequence-editor handoff identity mismatch")
	}
	todoPath, err := filepath.Abs(args[0])
	if err != nil || filepath.Base(todoPath) != "git-rebase-todo" || !pathWithin(manifest.GitDir, todoPath) {
		return errors.New("sequence-editor target is outside the active repository rebase metadata")
	}
	plan, err := os.ReadFile(manifest.PlanPath)
	if err != nil {
		return fmt.Errorf("read sequence-editor plan: %w", err)
	}
	digest := sha256.Sum256(plan)
	if hex.EncodeToString(digest[:]) != manifest.PlanSHA256 {
		return errors.New("sequence-editor plan integrity check failed")
	}
	temporary, err := os.CreateTemp(filepath.Dir(todoPath), ".gitwatch-todo-*.tmp")
	if err != nil {
		return fmt.Errorf("create sequence-editor replacement: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("protect sequence-editor replacement: %w", err)
	}
	if _, err := temporary.Write(plan); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write sequence-editor replacement: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync sequence-editor replacement: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close sequence-editor replacement: %w", err)
	}
	if err := os.Rename(temporaryPath, todoPath); err != nil {
		return fmt.Errorf("install sequence-editor plan: %w", err)
	}
	return nil
}

func pathWithin(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
