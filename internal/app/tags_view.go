package app

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/history"
	"github.com/sphireinc/git-watch/internal/platform"
	"github.com/sphireinc/git-watch/internal/tags"
	"github.com/sphireinc/git-watch/internal/worktrees"
)

func (m Model) selectedTag() (tags.Tag, bool) {
	rows := m.filteredTags()
	if m.TagsSelected < 0 || m.TagsSelected >= len(rows) {
		return tags.Tag{}, false
	}
	return rows[m.TagsSelected], true
}

func (m Model) createTag() tea.Cmd {
	request := tags.CreateRequest{
		Name:    strings.TrimSpace(m.TagCreateName),
		Target:  strings.TrimSpace(m.TagCreateTarget),
		Message: m.TagCreateMessage,
		Kind:    m.TagCreateKind,
	}
	generation := m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := tags.Create(m.commandContext(), runner, request)
		return TagMutationFinishedMsg{Generation: generation, Operation: "created", Name: request.Name, Err: err}
	}
}

func (m Model) deleteSelectedTag() tea.Cmd {
	name := m.TagDeleteTarget
	confirmed := m.TagDeleteInput
	generation := m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		_, err := tags.Delete(m.commandContext(), runner, name, confirmed)
		return TagMutationFinishedMsg{Generation: generation, Operation: "deleted", Name: name, Err: err}
	}
}

func (m *Model) resetTagMutation() {
	m.TagCreateMode, m.TagCreateKind, m.TagCreateName, m.TagCreateTarget = "", "", "", ""
	m.TagCreateMessage, m.TagCreateInput = "", ""
	m.TagDeleteMode, m.TagDeleteTarget, m.TagDeleteInput = false, "", ""
}

func (m *Model) startTagCreation(kind tags.CreateKind) {
	m.resetTagMutation()
	m.TagCreateKind, m.TagCreateMode, m.TagCreateInput = kind, "name", ""
	m.Status = string(kind) + " tag name: "
}

func (m *Model) updateTagMutationKey(key string) tea.Cmd {
	if m.TagCreateMode == "" && !m.TagDeleteMode {
		return nil
	}
	if key == "esc" {
		m.resetTagMutation()
		m.Status = "tag mutation cancelled"
		return nil
	}
	if key == "backspace" {
		m.TagCreateInput = removeLastRune(m.TagCreateInput)
		if m.TagDeleteMode {
			m.TagDeleteInput = removeLastRune(m.TagDeleteInput)
		}
	} else if key == "space" {
		if m.TagDeleteMode {
			m.TagDeleteInput += " "
		} else {
			m.TagCreateInput += " "
		}
	} else if key == "enter" {
		if m.TagDeleteMode {
			if strings.TrimSpace(m.TagDeleteInput) == "" {
				m.Status = "type the exact tag name to confirm deletion"
				return nil
			}
			m.TagDeleteMode, m.State, m.Status = false, StateOperationPending, "deleting tag"
			return m.deleteSelectedTag()
		}
		value := strings.TrimSpace(m.TagCreateInput)
		if value == "" {
			m.Status = "tag input is required"
			return nil
		}
		switch m.TagCreateMode {
		case "name":
			m.TagCreateName, m.TagCreateTarget, m.TagCreateInput, m.TagCreateMode = value, "", "HEAD", "target"
			m.Status = "tag target (default HEAD): HEAD"
			return nil
		case "target":
			m.TagCreateTarget, m.TagCreateInput = value, ""
			if m.TagCreateKind == tags.CreateLightweight {
				m.TagCreateMode, m.State, m.Status = "", StateOperationPending, "creating lightweight tag"
				return m.createTag()
			}
			m.TagCreateMode = "message"
			m.Status = "tag message: "
			return nil
		case "message":
			m.TagCreateMessage, m.TagCreateInput, m.TagCreateMode = value, "", ""
			m.State, m.Status = StateOperationPending, "creating "+string(m.TagCreateKind)+" tag"
			return m.createTag()
		}
	} else if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
		if m.TagDeleteMode {
			m.TagDeleteInput += key
		} else {
			m.TagCreateInput += key
		}
	} else {
		return nil
	}
	if m.TagDeleteMode {
		m.Status = "type " + platform.SafeText(m.TagDeleteTarget) + " to confirm deletion: " + platform.SafeText(m.TagDeleteInput)
	} else {
		m.Status = "tag " + m.TagCreateMode + ": " + platform.SafeText(m.TagCreateInput)
	}
	return nil
}

func (m *Model) verifySelectedTag() tea.Cmd {
	selected, ok := m.selectedTag()
	if !ok {
		m.Status = "select a tag first"
		return nil
	}
	if selected.Kind == tags.Lightweight {
		m.setTagSignature(selected.Name, tags.SignatureUnsigned)
		m.Status = "lightweight tag has no signature: " + platform.SafeText(selected.Name)
		return nil
	}
	m.TagSignatureChecking = selected.Name
	m.State, m.Status = StateOperationPending, "verifying tag "+platform.SafeText(selected.Name)
	name, generation := selected.Name, m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		state, err := tags.Verify(m.commandContext(), runner, name)
		return TagSignatureReadyMsg{Generation: generation, Name: name, State: state, Err: err}
	}
}

func (m *Model) setTagSignature(name string, state tags.SignatureState) {
	for i := range m.TagSnapshot.Tags {
		if m.TagSnapshot.Tags[i].Name == name {
			m.TagSnapshot.Tags[i].Signature = state
			return
		}
	}
}

func (m Model) inspectSelectedTag() tea.Cmd {
	selected, ok := m.selectedTag()
	if !ok {
		return nil
	}
	commit := history.Commit{SHA: selected.TargetID, Short: selected.TargetID, Subject: selected.Message, Refs: []string{selected.Name}}
	runner := git.NewRunner(m.Discovery.Root)
	return func() tea.Msg {
		inspector, err := history.InspectPath(m.commandContext(), runner, selected.TargetID, "", "")
		inspector.Commit = commit
		return HistoryInspectorReadyMsg{Inspector: inspector, Err: err}
	}
}

func (m Model) checkoutSelectedTag() tea.Cmd {
	selected, ok := m.selectedTag()
	if !ok {
		return nil
	}
	name, generation := selected.Name, m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.commandContext()
	return func() tea.Msg {
		_, err := history.CheckoutCommit(ctx, runner, selected.TargetID)
		return TagCheckoutFinishedMsg{Generation: generation, Name: name, Err: err}
	}
}

func (m Model) compareSelectedTag() tea.Cmd {
	selected, ok := m.selectedTag()
	if !ok {
		return nil
	}
	name, generation := selected.Name, m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.commandContext()
	return func() tea.Msg {
		inspector, err := history.InspectPath(ctx, runner, selected.TargetID, "HEAD", "")
		return TagCompareReadyMsg{Generation: generation, Name: name, Text: inspector.Diff, Err: err}
	}
}

func (m Model) addSelectedTagWorktree() tea.Cmd {
	selected, ok := m.selectedTag()
	path := strings.TrimSpace(m.TagWorktreePath)
	if !ok || path == "" {
		return nil
	}
	name, generation := selected.Name, m.repositoryGeneration
	runner := git.NewRunner(m.Discovery.Root)
	ctx := m.commandContext()
	return func() tea.Msg {
		_, err := worktrees.AddWithCommit(ctx, runner, path, "", selected.TargetID)
		return TagWorktreeFinishedMsg{Generation: generation, Name: name, Path: path, Err: err}
	}
}

func (m *Model) updateTagWorktreeKey(key string) tea.Cmd {
	switch key {
	case "esc":
		m.TagWorktreeMode, m.TagWorktreePath = false, ""
		m.Status = "tag worktree creation cancelled"
	case "backspace":
		m.TagWorktreePath = removeLastRune(m.TagWorktreePath)
	case "enter":
		if strings.TrimSpace(m.TagWorktreePath) == "" {
			m.Status = "worktree path is required"
		} else {
			m.TagWorktreeMode, m.State, m.Status = false, StateOperationPending, "creating worktree from tag"
			return m.addSelectedTagWorktree()
		}
	case "space":
		m.TagWorktreePath += " "
	default:
		if len([]rune(key)) == 1 && !strings.ContainsAny(key, "\r\n\x00") {
			m.TagWorktreePath += key
		}
	}
	if m.TagWorktreeMode {
		m.Status = "tag worktree path: " + platform.SafeText(m.TagWorktreePath)
	}
	return nil
}

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
		lines = append(lines, prefix+platform.SafeText(tag.Name)+" · "+string(tag.Kind)+" · "+platform.SafeText(tag.TargetID)+" · signature: "+string(tag.Signature)+" · "+remote)
	}
	if len(rows) == 0 {
		lines = append(lines, "  No matching tags")
	}
	return strings.Join(lines, "\n")
}
