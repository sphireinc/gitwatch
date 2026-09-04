package history

import (
	"errors"
	"fmt"
)

// Selection is an immutable, repository-scoped commit basket. SHAs are kept
// in Git application order (oldest first); callers may render them differently.
type Selection struct {
	repository string
	ref        string
	generation uint64
	shas       []string
}

// NewSelection creates an empty basket for one repository and history scope.
func NewSelection(repository, ref string, generation uint64) (Selection, error) {
	if repository == "" {
		return Selection{}, errors.New("commit selection repository is required")
	}
	return Selection{repository: repository, ref: ref, generation: generation}, nil
}

func (s Selection) Repository() string { return s.repository }
func (s Selection) Ref() string        { return s.ref }
func (s Selection) Generation() uint64 { return s.generation }
func (s Selection) SHAs() []string     { return append([]string(nil), s.shas...) }
func (s Selection) Count() int         { return len(s.shas) }

// InScope rejects a basket from another repository or an older history load.
func (s Selection) InScope(repository, ref string, generation uint64) error {
	if s.repository != repository || s.ref != ref {
		return errors.New("commit selection belongs to a different repository or ref")
	}
	if generation != 0 && s.generation != 0 && s.generation > generation {
		return errors.New("commit selection belongs to a newer history generation")
	}
	return nil
}

// Toggle adds or removes one SHA while preserving application order.
func (s Selection) Toggle(sha string) (Selection, error) {
	if sha == "" {
		return Selection{}, errors.New("commit selection SHA is required")
	}
	result := s.clone()
	for index, existing := range result.shas {
		if existing == sha {
			result.shas = append(result.shas[:index], result.shas[index+1:]...)
			return result, nil
		}
	}
	result.shas = append(result.shas, sha)
	return result, nil
}

// SelectRange adds a display-order range and normalizes it to oldest-first
// application order. The input is typically history's newest-first list.
func (s Selection) SelectRange(commits []Commit, start, end int) (Selection, error) {
	if start < 0 || end < 0 || start >= len(commits) || end >= len(commits) {
		return Selection{}, fmt.Errorf("commit selection range %d..%d is out of bounds", start, end)
	}
	if start > end {
		start, end = end, start
	}
	result := s.clone()
	for index := end; index >= start; index-- {
		sha := commits[index].SHA
		if sha == "" {
			return Selection{}, errors.New("commit selection contains an empty SHA")
		}
		if !contains(result.shas, sha) {
			result.shas = append(result.shas, sha)
		}
	}
	return result, nil
}

// Clear returns an empty basket in the same scope.
func (s Selection) Clear() Selection { s.shas = nil; return s }

func (s Selection) clone() Selection { s.shas = append([]string(nil), s.shas...); return s }
func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
