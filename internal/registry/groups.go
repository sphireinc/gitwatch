package registry

import "sort"

func Groups(repositories []Repository) []string {
	seen := make(map[string]bool)
	var groups []string
	for _, repository := range repositories {
		for _, group := range repository.Groups {
			if group != "" && !seen[group] {
				seen[group] = true
				groups = append(groups, group)
			}
		}
	}
	sort.Strings(groups)
	return groups
}

func InGroup(repositories []Repository, group string) []Repository {
	var filtered []Repository
	for _, repository := range repositories {
		for _, candidate := range repository.Groups {
			if candidate == group {
				filtered = append(filtered, repository)
				break
			}
		}
	}
	return Order(filtered)
}

func SetFavorite(repositories []Repository, path string, favorite bool) []Repository {
	updated := clone(repositories)
	for i := range updated {
		if updated[i].Path == path {
			updated[i].Favorite = favorite
		}
	}
	return Order(updated)
}

func AddToGroup(repositories []Repository, path, group string) []Repository {
	updated := clone(repositories)
	for i := range updated {
		if updated[i].Path != path || group == "" {
			continue
		}
		for _, candidate := range updated[i].Groups {
			if candidate == group {
				return Order(updated)
			}
		}
		updated[i].Groups = append(updated[i].Groups, group)
	}
	return Order(updated)
}

func Order(repositories []Repository) []Repository {
	ordered := clone(repositories)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Favorite != ordered[j].Favorite {
			return ordered[i].Favorite
		}
		if !ordered[i].LastOpened.Equal(ordered[j].LastOpened) {
			return ordered[i].LastOpened.After(ordered[j].LastOpened)
		}
		return ordered[i].Name < ordered[j].Name
	})
	return ordered
}

func clone(repositories []Repository) []Repository {
	cloned := make([]Repository, len(repositories))
	copy(cloned, repositories)
	for i := range cloned {
		cloned[i].Groups = append([]string(nil), cloned[i].Groups...)
	}
	return cloned
}
