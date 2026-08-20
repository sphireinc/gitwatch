package provider

import (
	"encoding/json"
	"errors"
	"time"
)

type CheckRun struct {
	Name        string
	Status      string
	Conclusion  string
	URL         string
	StartedAt   time.Time
	CompletedAt time.Time
	Failure     string
}

func (r CheckRun) Duration() time.Duration {
	if r.StartedAt.IsZero() {
		return 0
	}
	end := r.CompletedAt
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(r.StartedAt)
}

type ChecksSnapshot struct {
	Runs    []CheckRun
	Passing int
	Failing int
	Pending int
}

func ParseChecks(data []byte) (ChecksSnapshot, error) {
	var response struct {
		CheckRuns []struct {
			Name        string     `json:"name"`
			Status      string     `json:"status"`
			Conclusion  string     `json:"conclusion"`
			URL         string     `json:"html_url"`
			StartedAt   *time.Time `json:"started_at"`
			CompletedAt *time.Time `json:"completed_at"`
			Output      struct {
				Title   string `json:"title"`
				Summary string `json:"summary"`
			} `json:"output"`
		} `json:"check_runs"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return ChecksSnapshot{}, err
	}
	if response.CheckRuns == nil {
		return ChecksSnapshot{}, errors.New("invalid checks response")
	}
	snapshot := ChecksSnapshot{Runs: make([]CheckRun, 0, len(response.CheckRuns))}
	for _, raw := range response.CheckRuns {
		run := CheckRun{Name: raw.Name, Status: raw.Status, Conclusion: raw.Conclusion, URL: raw.URL, Failure: raw.Output.Summary}
		if run.Failure == "" {
			run.Failure = raw.Output.Title
		}
		if raw.StartedAt != nil {
			run.StartedAt = raw.StartedAt.UTC()
		}
		if raw.CompletedAt != nil {
			run.CompletedAt = raw.CompletedAt.UTC()
		}
		snapshot.Runs = append(snapshot.Runs, run)
		if run.Status != "completed" {
			snapshot.Pending++
		} else if run.Conclusion == "success" || run.Conclusion == "neutral" || run.Conclusion == "skipped" {
			snapshot.Passing++
		} else {
			snapshot.Failing++
		}
	}
	return snapshot, nil
}
