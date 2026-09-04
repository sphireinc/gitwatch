// Package multirepo contains explicit, repository-scoped batch gitignore workflows.
package multirepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/sphireinc/git-watch/internal/gitignore/catalog"
	"github.com/sphireinc/git-watch/internal/gitignore/domain"
	"github.com/sphireinc/git-watch/internal/gitignore/manage"
)

type Repository struct {
	ID   domain.RepositoryID
	Root string
}
type PlanResult struct {
	Repository Repository
	Plan       domain.MutationPlan
	Preview    manage.Preview
	Status     string
	Skipped    bool
	Err        error
}
type ApplyResult struct {
	Repository Repository
	Status     string
	Err        error
}
type Refresh func(context.Context, Repository) error

// PlanAdd reads only explicitly selected repositories and creates one plan per repository.
func PlanAdd(ctx context.Context, repositories []Repository, cat *catalog.Catalog, ids []domain.TemplateID) []PlanResult {
	results := make([]PlanResult, len(repositories))
	for i, repository := range repositories {
		results[i].Repository = repository
		select {
		case <-ctx.Done():
			results[i].Err = ctx.Err()
			continue
		default:
		}
		path := filepath.Join(repository.Root, ".gitignore")
		info, statErr := os.Lstat(path)
		existed := statErr == nil
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			results[i].Err = statErr
			continue
		}
		if existed && info.Mode()&os.ModeSymlink != 0 {
			results[i].Skipped, results[i].Status = true, "symlink"
			continue
		}
		if existed && info.Mode().Perm()&0222 == 0 {
			results[i].Skipped, results[i].Status = true, "read-only"
			continue
		}
		data := []byte(nil)
		permissions := uint32(0644)
		if existed {
			data, statErr = os.ReadFile(path)
			permissions = uint32(info.Mode().Perm())
		}
		if existed && statErr != nil {
			results[i].Err = statErr
			continue
		}
		snapshot, err := domain.NewDocumentSnapshot(repository.ID, repository.Root, ".gitignore", data, permissions)
		if err != nil {
			results[i].Err = err
			continue
		}
		if !existed {
			results[i].Plan, err = manage.PlanCreateTemplates(snapshot, cat, ids)
		} else {
			results[i].Plan, err = manage.PlanAddTemplates(snapshot, cat, ids)
		}
		if err != nil {
			results[i].Err = err
			continue
		}
		results[i].Preview = manage.PreviewPlan(results[i].Plan)
	}
	return results
}

// Apply runs explicitly planned repositories with bounded concurrency. A failed repository never prevents unrelated plans.
func Apply(ctx context.Context, plans []PlanResult, workers int, refresh Refresh) []ApplyResult {
	if workers < 1 {
		workers = 1
	}
	results := make([]ApplyResult, len(plans))
	jobs := make(chan int)
	var group sync.WaitGroup
	for n := 0; n < workers; n++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				plan := plans[index]
				result := ApplyResult{Repository: plan.Repository}
				if plan.Skipped {
					result.Status, result.Err = "skipped", errors.New(plan.Status)
					results[index] = result
					continue
				}
				if plan.Err != nil {
					result.Status, result.Err = "failed", plan.Err
					results[index] = result
					continue
				}
				select {
				case <-ctx.Done():
					result.Status, result.Err = "failed", ctx.Err()
				default:
					result.Err = manage.Apply(plan.Plan)
					if result.Err == nil && refresh != nil {
						result.Err = refresh(ctx, plan.Repository)
					}
					if result.Err == nil {
						result.Status = "succeeded"
					} else {
						result.Status = "failed"
					}
				}
				results[index] = result
			}
		}()
	}
send:
	for i := range plans {
		select {
		case jobs <- i:
		case <-ctx.Done():
			break send
		}
	}
	close(jobs)
	group.Wait()
	return results
}
