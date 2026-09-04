package manage

import (
	"errors"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
)

// PlanCreateTemplates is the creation-only variant of PlanAddTemplates. It
// refuses an existing target so a stale no-file wizard can never turn into an
// append operation or overwrite another process's file.
func PlanCreateTemplates(snapshot domain.DocumentSnapshot, cat *catalog.Catalog, ids []domain.TemplateID) (domain.MutationPlan, error) {
	if snapshot.Path != ".gitignore" || len(snapshot.Bytes) != 0 || snapshot.SHA256 != emptyHash() {
		return domain.MutationPlan{}, domain.ErrConcurrentModification
	}
	plan, err := PlanAddTemplates(snapshot, cat, ids)
	if err != nil {
		return domain.MutationPlan{}, err
	}
	plan.Kind = domain.MutationCreate
	return plan, nil
}

// Create applies a no-file creation plan. Apply performs the final absence
// and race checks; callers should request the authoritative Git refresh after
// it returns successfully.
func Create(plan domain.MutationPlan) error {
	if plan.Kind != domain.MutationCreate && plan.Kind != domain.MutationAppend {
		return errors.New("gitignore creation requires a create plan")
	}
	return Apply(plan)
}

func emptyHash() string { return hash(nil) }
