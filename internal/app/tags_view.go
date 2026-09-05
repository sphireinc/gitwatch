package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/tags"
)

func (m Model) filteredTags() []tags.Tag {
	rows := append([]tags.Tag(nil), m.TagSnapshot.Tags...)
	query := strings.ToLower(strings.TrimSpace(m.TagsFilter))
	if query != "" {
		rows = rows[:0]
		for _, tag := range m.TagSnapshot.Tags {
			if strings.Contains(strings.ToLower(tag.Name), query) || strings.Contains(strings.ToLower(tag.TargetID), query) || strings.Contains(strings.ToLower(tag.Message), query) {
				rows = append(rows, tag)
			}
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		left, right := rows[i], rows[j]
		var less bool
		switch m.TagsSort {
		case "target":
			less = left.TargetID < right.TargetID
		case "date":
			less = left.TaggedAt.Before(right.TaggedAt)
		default:
			less = strings.ToLower(left.Name) < strings.ToLower(right.Name)
		}
		if m.TagsSortDesc {
			return !less && left.Name != right.Name
		}
		return less
	})
	return rows
}

func (m *Model) moveTags(delta int) {
	rows := m.filteredTags()
	m.TagsSelected += delta
	if m.TagsSelected < 0 {
		m.TagsSelected = 0
	}
	if m.TagsSelected >= len(rows) {
		m.TagsSelected = len(rows) - 1
	}
}

func (m *Model) cycleTagsSort() {
	keys := []string{"name", "target", "date"}
	index := 0
	for i, key := range keys {
		if key == m.TagsSort {
			index = i
			break
		}
	}
	if index == len(keys)-1 {
		if !m.TagsSortDesc {
			m.TagsSortDesc = true
			return
		}
		m.TagsSortDesc = false
		index = 0
	} else {
		index++
	}
	m.TagsSort = keys[index]
}

func (m *Model) updateTagsFilter(key string) {
	switch key {
	case "esc":
		m.TagsFilterMode = false
		m.Status = "tag filter cancelled"
	case "backspace":
		m.TagsFilter = removeLastRune(m.TagsFilter)
	case "enter":
		m.TagsFilterMode = false
		m.TagsSelected = 0
		m.Status = fmt.Sprintf("tag filter applied: %d match(es)", len(m.filteredTags()))
	case "space":
		m.TagsFilter += " "
	default:
		if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
			m.TagsFilter += key
		}
	}
	if m.TagsFilterMode {
		m.Status = "tag filter: " + platform.SafeText(m.TagsFilter)
	}
}

func (m Model) tagsView() string {
	if m.TagsLoading {
		return "Tags\n\nLoading tags…"
	}
	if m.TagsErr != nil {
		return "Tags\n\nError: " + platform.SafeText(m.TagsErr.Error())
	}
	rows := m.filteredTags()
	header := fmt.Sprintf("Tags · %d", len(rows))
	if m.TagsFilter != "" {
		header += " · filter: " + platform.SafeText(m.TagsFilter)
	}
	header += " · sort: " + m.TagsSort
	if m.TagsSortDesc {
		header += " desc"
	}
	lines := []string{header}
	for i, tag := range rows {
		prefix := "  "
		if i == m.TagsSelected {
			prefix = "> "
		}
		remote := "local"
		if tag.RemotePresence == tags.RemotePresent {
			remote = "remote: " + strings.Join(tag.RemoteNames, ",")
		}
		lines = append(lines, prefix+platform.SafeText(tag.Name)+" · "+string(tag.Kind)+" · "+platform.SafeText(tag.TargetID)+" · "+remote)
	}
	if len(rows) == 0 {
		lines = append(lines, "  No matching tags")
	}
	return strings.Join(lines, "\n")
}
