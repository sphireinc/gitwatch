package manage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

var ErrUndoConflict = errors.New("gitignore changed after the transaction and cannot be undone safely")

type OperationRecord struct {
	Repository   domain.RepositoryID
	Root         string
	Path         string
	Kind         domain.MutationKind
	Selected     []domain.TemplateID
	BeforeSHA256 string
	AfterSHA256  string
	BeforeBytes  []byte
	Success      bool
}

func ApplyTransaction(plan domain.MutationPlan) (OperationRecord, error) {
	record := OperationRecord{Repository: plan.Repository, Root: plan.Root, Path: plan.Path, Kind: plan.Kind, Selected: append([]domain.TemplateID(nil), plan.Selected...), BeforeSHA256: plan.BeforeSHA256, BeforeBytes: append([]byte(nil), plan.BeforeBytes...)}
	if err := Apply(plan); err != nil {
		return record, err
	}
	record.AfterSHA256 = hashBytes(plan.ResultBytes)
	record.Success = true
	return record, nil
}

func Undo(record OperationRecord) error {
	if !record.Success || record.Root == "" || record.Path != ".gitignore" {
		return ErrUndoConflict
	}
	path := filepath.Join(record.Root, record.Path)
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return domain.ErrUnsafeTarget
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if hashBytes(current) != record.AfterSHA256 {
		return ErrUndoConflict
	}
	snapshot, err := domain.NewDocumentSnapshot(record.Repository, record.Root, record.Path, current, uint32(info.Mode().Perm()))
	if err != nil {
		return err
	}
	plan, err := domain.NewMutationPlan(snapshot, domain.MutationUpdate, record.Selected, []domain.Edit{{Start: 0, End: len(current), Replacement: append([]byte(nil), record.BeforeBytes...)}}, record.BeforeBytes, nil)
	if err != nil {
		return err
	}
	return Apply(plan)
}

func hashBytes(value []byte) string { sum := sha256.Sum256(value); return hex.EncodeToString(sum[:]) }

func (r OperationRecord) Summary() string {
	return fmt.Sprintf("%s %s %s", r.Repository, r.Kind, r.Path)
}
