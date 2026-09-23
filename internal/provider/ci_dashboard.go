package provider

import (
	"context"
	"sync"
)

const (
	MaxDashboardCIRepositories = 20
	DashboardCIWorkers         = 2
)

// CISummaryRequest is one repository-scoped CI lookup for a dashboard.
type CISummaryRequest struct {
	Key string
	Ref string
}

// CISummary is the bounded, render-ready result for one repository lookup.
type CISummary struct {
	Key       string
	State     State
	Attention string
	Stale     bool
	Skipped   bool
}

// LoadCISummaries fetches at most twenty repositories with a bounded worker
// pool and returns results in request order. It is designed to run as
// independent background provider work beside local Git status refreshes.
func LoadCISummaries(ctx context.Context, requests []CISummaryRequest, workers int, fetch func(context.Context, CISummaryRequest) CISummary) []CISummary {
	if len(requests) > MaxDashboardCIRepositories {
		requests = requests[:MaxDashboardCIRepositories]
	}
	if len(requests) == 0 || fetch == nil {
		return nil
	}
	if workers < 1 {
		workers = 1
	}
	if workers > DashboardCIWorkers {
		workers = DashboardCIWorkers
	}
	if workers > len(requests) {
		workers = len(requests)
	}
	type indexed struct {
		index int
		value CISummary
	}
	jobs := make(chan int, len(requests))
	results := make(chan indexed, len(requests))
	for index := range requests {
		jobs <- index
	}
	close(jobs)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range jobs {
				if ctx.Err() != nil {
					return
				}
				value := fetch(ctx, requests[index])
				if value.Key == "" {
					value.Key = requests[index].Key
				}
				results <- indexed{index: index, value: value}
			}
		}()
	}
	group.Wait()
	close(results)
	values := make([]CISummary, len(requests))
	seen := make([]bool, len(requests))
	for result := range results {
		values[result.index] = result.value
		seen[result.index] = true
	}
	ordered := make([]CISummary, 0, len(requests))
	for index, value := range values {
		if seen[index] {
			ordered = append(ordered, value)
		}
	}
	return ordered
}
