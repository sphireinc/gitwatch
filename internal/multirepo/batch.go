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

type Action string

const (
	ActionFetch Action = "fetch"
	ActionPull  Action = "pull"
)

type Request struct {
	Repository Repository
	Remote     string
	Branch     string
	Strategy   string
	Action     Action
}

type Result struct {
	Request Request
	Status  string
	Err     error
}

func (r Request) Validate() error {
	if r.Repository.ID == "" || r.Repository.Root == "" {
		return errors.New("batch request requires repository identity and root")
	}
	if r.Remote == "" {
		return errors.New("batch request requires an explicit remote")
	}
	switch r.Action {
	case ActionFetch:
		if r.Branch != "" || r.Strategy != "" {
			return errors.New("fetch request cannot include branch or strategy")
		}
	case ActionPull:
		if r.Branch == "" {
			return errors.New("pull request requires a branch")
		}
		if r.Strategy != "ff-only" && r.Strategy != "merge" && r.Strategy != "rebase" {
			return errors.New("pull request requires an explicit strategy")
		}
	default:
		return errors.New("unsupported batch action")
	}
	return nil
}

// Run executes explicitly planned repository operations with bounded
// concurrency. The callback owns the typed Git/provider operation; one error
// is recorded and does not prevent unrelated requests from running.
func Run(ctx context.Context, requests []Request, workers int, execute func(context.Context, Request) error) []Result {
	if workers < 1 {
		workers = 1
	}
	results := make([]Result, len(requests))
	jobs := make(chan int)
	var group sync.WaitGroup
	for n := 0; n < workers; n++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				request := requests[index]
				result := Result{Request: request, Status: "queued"}
				if err := request.Validate(); err != nil {
					result.Status, result.Err = "skipped", err
					results[index] = result
					continue
				}
				select {
				case <-ctx.Done():
					result.Status, result.Err = "cancelled", ctx.Err()
				default:
					result.Status = "running"
					if execute == nil {
						result.Err = errors.New("batch executor is required")
					} else {
						result.Err = execute(ctx, request)
					}
					if result.Err == nil {
						result.Status = "succeeded"
					} else if errors.Is(result.Err, context.Canceled) || errors.Is(result.Err, context.DeadlineExceeded) {
						result.Status = "cancelled"
					} else {
						result.Status = "failed"
					}
				}
				results[index] = result
			}
		}()
	}
	for index := range requests {
		select {
		case jobs <- index:
		case <-ctx.Done():
			for remaining := index; remaining < len(requests); remaining++ {
				results[remaining] = Result{Request: requests[remaining], Status: "cancelled", Err: ctx.Err()}
			}
			index = len(requests)
		}
		if index == len(requests) {
			break
		}
	}
	close(jobs)
	group.Wait()
	return results
}

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
