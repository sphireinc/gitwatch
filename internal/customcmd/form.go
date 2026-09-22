package customcmd

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// PromptKind identifies the input control required before a command runs.
type PromptKind string

const (
	PromptText        PromptKind = "text"
	PromptSecret      PromptKind = "secret"
	PromptConfirm     PromptKind = "confirm"
	PromptSelect      PromptKind = "select"
	PromptMultiSelect PromptKind = "multi-select"
)

// Prompt is the serializable, host-rendered schema for one custom-command
// input. Values are returned separately from the command definition so a
// canceled or incomplete form can never start a process.
type Prompt struct {
	ID            string     `json:"id"`
	Label         string     `json:"label"`
	Kind          PromptKind `json:"kind"`
	Required      bool       `json:"required,omitempty"`
	Pattern       string     `json:"pattern,omitempty"`
	Options       []string   `json:"options,omitempty"`
	OptionsSource string     `json:"options_source,omitempty"`
	Default       string     `json:"default,omitempty"`
}

// FormEvent describes the result of handling one key in a form.
type FormEvent uint8

const (
	FormChanged FormEvent = iota
	FormSubmitted
	FormCancelled
)

// Form is a small Bubble Tea-independent state machine. The app owns key and
// rendering policy; this type owns validation, focus, typed control behavior,
// and the invariant that values exist only after submit.
type Form struct {
	prompts  []Prompt
	values   map[string]string
	selected map[string]map[string]bool
	index    int
	input    string
	cursor   int
}

// NewForm validates and initializes a form without executing anything.
func NewForm(prompts []Prompt) (Form, error) {
	if len(prompts) == 0 {
		return Form{}, fmt.Errorf("custom command form requires at least one prompt")
	}
	seen := make(map[string]struct{}, len(prompts))
	copyPrompts := make([]Prompt, len(prompts))
	for index, prompt := range prompts {
		if strings.TrimSpace(prompt.ID) == "" || strings.TrimSpace(prompt.Label) == "" {
			return Form{}, fmt.Errorf("prompt %d requires an id and label", index)
		}
		if _, exists := seen[prompt.ID]; exists {
			return Form{}, fmt.Errorf("duplicate prompt %q", prompt.ID)
		}
		seen[prompt.ID] = struct{}{}
		if err := validatePrompt(prompt); err != nil {
			return Form{}, err
		}
		copyPrompts[index] = Prompt{ID: prompt.ID, Label: prompt.Label, Kind: prompt.Kind, Required: prompt.Required, Pattern: prompt.Pattern, Options: append([]string(nil), prompt.Options...), OptionsSource: prompt.OptionsSource, Default: prompt.Default}
	}
	form := Form{prompts: copyPrompts, values: make(map[string]string), selected: make(map[string]map[string]bool)}
	form.loadCurrent()
	return form, nil
}

func validatePrompt(prompt Prompt) error {
	switch prompt.Kind {
	case PromptText, PromptSecret, PromptConfirm:
		if len(prompt.Options) != 0 {
			return fmt.Errorf("prompt %q does not accept options", prompt.ID)
		}
	case PromptSelect, PromptMultiSelect:
		if len(prompt.Options) == 0 && prompt.OptionsSource == "" {
			return fmt.Errorf("prompt %q requires options", prompt.ID)
		}
		seen := make(map[string]struct{}, len(prompt.Options))
		for _, option := range prompt.Options {
			if option == "" {
				return fmt.Errorf("prompt %q contains an empty option", prompt.ID)
			}
			if _, exists := seen[option]; exists {
				return fmt.Errorf("prompt %q contains duplicate option %q", prompt.ID, option)
			}
			seen[option] = struct{}{}
		}
	default:
		return fmt.Errorf("prompt %q has unknown kind %q", prompt.ID, prompt.Kind)
	}
	if prompt.OptionsSource != "" {
		switch prompt.OptionsSource {
		case "branches", "remotes", "tags", "commits", "paths":
		default:
			return fmt.Errorf("prompt %q has unknown options source %q", prompt.ID, prompt.OptionsSource)
		}
	}
	if prompt.Pattern != "" {
		if _, err := regexp.Compile(prompt.Pattern); err != nil {
			return fmt.Errorf("prompt %q has invalid pattern: %w", prompt.ID, err)
		}
	}
	return nil
}

// ResolvePrompts fills dynamic option sources from already-loaded repository
// state. It never runs Git or a provider request while a form is opening.
func ResolvePrompts(prompts []Prompt, ctx Context) ([]Prompt, error) {
	resolved := make([]Prompt, len(prompts))
	for index, prompt := range prompts {
		resolved[index] = prompt
		if prompt.OptionsSource == "" {
			continue
		}
		options := append([]string(nil), prompt.Options...)
		options = append(options, ctx.OptionValues[prompt.OptionsSource]...)
		seen := make(map[string]struct{}, len(options))
		prompt.Options = options[:0]
		for _, option := range options {
			if option != "" {
				if _, exists := seen[option]; !exists {
					seen[option] = struct{}{}
					prompt.Options = append(prompt.Options, option)
				}
			}
		}
		if len(prompt.Options) == 0 {
			return nil, fmt.Errorf("prompt %q has no loaded options", prompt.ID)
		}
		resolved[index] = prompt
	}
	if _, err := NewForm(resolved); err != nil {
		return nil, err
	}
	return resolved, nil
}

func (f *Form) loadCurrent() {
	if f.index < 0 || f.index >= len(f.prompts) {
		return
	}
	prompt := f.prompts[f.index]
	f.input = f.values[prompt.ID]
	f.cursor = 0
	if prompt.Kind == PromptSelect || prompt.Kind == PromptMultiSelect {
		for index, option := range prompt.Options {
			if option == f.input {
				f.cursor = index
				break
			}
		}
	}
}

// Current returns the focused prompt.
func (f Form) Current() (Prompt, bool) {
	if f.index < 0 || f.index >= len(f.prompts) {
		return Prompt{}, false
	}
	return f.prompts[f.index], true
}

// Handle applies one normalized key. Text input accepts one-rune strings;
// navigation uses up/down, space, enter, backspace, and esc.
func (f *Form) Handle(key string) (FormEvent, error) {
	if key == "esc" {
		return FormCancelled, nil
	}
	prompt, ok := f.Current()
	if !ok {
		return FormSubmitted, nil
	}
	switch prompt.Kind {
	case PromptText, PromptSecret:
		switch key {
		case "backspace":
			if f.input != "" {
				runes := []rune(f.input)
				f.input = string(runes[:len(runes)-1])
			}
		case "enter":
			if err := f.acceptCurrent(); err != nil {
				return FormChanged, err
			}
			return f.advance(), nil
		default:
			if len([]rune(key)) == 1 && key != "\x00" && key != "\r" && key != "\n" {
				f.input += key
			}
		}
	case PromptConfirm:
		switch strings.ToLower(key) {
		case "y", "yes", "enter":
			f.input = "true"
			f.values[prompt.ID] = f.input
			return f.advance(), nil
		case "n", "no":
			f.input = "false"
			return FormCancelled, nil
		}
	case PromptSelect:
		switch key {
		case "up", "k":
			f.cursor = (f.cursor - 1 + len(prompt.Options)) % len(prompt.Options)
		case "down", "j":
			f.cursor = (f.cursor + 1) % len(prompt.Options)
		case "enter":
			f.input = prompt.Options[f.cursor]
			return f.advance(), nil
		}
	case PromptMultiSelect:
		if f.selected[prompt.ID] == nil {
			f.selected[prompt.ID] = make(map[string]bool)
		}
		switch key {
		case "up", "k":
			f.cursor = (f.cursor - 1 + len(prompt.Options)) % len(prompt.Options)
		case "down", "j":
			f.cursor = (f.cursor + 1) % len(prompt.Options)
		case "space":
			option := prompt.Options[f.cursor]
			f.selected[prompt.ID][option] = !f.selected[prompt.ID][option]
		case "enter":
			selected := make([]string, 0, len(prompt.Options))
			for _, option := range prompt.Options {
				if f.selected[prompt.ID][option] {
					selected = append(selected, option)
				}
			}
			f.input = strings.Join(selected, ",")
			return f.advance(), nil
		}
	}
	return FormChanged, nil
}

func (f *Form) acceptCurrent() error {
	prompt, _ := f.Current()
	if prompt.Required && strings.TrimSpace(f.input) == "" {
		return fmt.Errorf("%s is required", prompt.Label)
	}
	if prompt.Pattern != "" && f.input != "" {
		matched, _ := regexp.MatchString(prompt.Pattern, f.input)
		if !matched {
			return fmt.Errorf("%s has an invalid value", prompt.Label)
		}
	}
	f.values[prompt.ID] = f.input
	return nil
}

func (f *Form) advance() FormEvent {
	prompt, _ := f.Current()
	if prompt.Kind != PromptConfirm {
		if err := f.acceptCurrent(); err != nil {
			return FormChanged
		}
	}
	f.index++
	if f.index >= len(f.prompts) {
		return FormSubmitted
	}
	f.loadCurrent()
	return FormChanged
}

// Values returns a copy only after all prompts have been accepted. It is safe
// to call while editing; incomplete values are intentionally not exposed.
func (f Form) Values() map[string]string {
	if f.index < len(f.prompts) {
		return nil
	}
	values := make(map[string]string, len(f.values))
	for key, value := range f.values {
		values[key] = value
	}
	return values
}

// Progress returns the one-based prompt position and total count.
func (f Form) Progress() (int, int) { return f.index + 1, len(f.prompts) }

// Input returns the focused editable value for host rendering. Secret
// prompts are masked without exposing their contents to the view layer.
func (f Form) Input() string {
	if prompt, ok := f.Current(); ok && prompt.Kind == PromptSecret {
		return strings.Repeat("•", len([]rune(f.input)))
	}
	return f.input
}

// Options returns the focused select options for a host renderer.
func (f Form) Options() []string {
	prompt, ok := f.Current()
	if !ok {
		return nil
	}
	return append([]string(nil), prompt.Options...)
}

// Cursor returns the zero-based focused option for select controls.
func (f Form) Cursor() int { return f.cursor }

// SelectedOptions returns the current multi-select values in declaration order.
func (f Form) SelectedOptions() []string {
	prompt, ok := f.Current()
	if !ok || prompt.Kind != PromptMultiSelect {
		return nil
	}
	selected := f.selected[prompt.ID]
	result := make([]string, 0, len(selected))
	for _, option := range prompt.Options {
		if selected[option] {
			result = append(result, option)
		}
	}
	return result
}

// RedactedValues returns submitted values with secret prompts replaced.
func (f Form) RedactedValues() map[string]string {
	values := f.Values()
	if values == nil {
		return nil
	}
	for _, prompt := range f.prompts {
		if prompt.Kind == PromptSecret && values[prompt.ID] != "" {
			values[prompt.ID] = "[redacted]"
		}
	}
	return values
}

// PromptIDs returns stable IDs for binding placeholders and tests.
func (f Form) PromptIDs() []string {
	ids := make([]string, 0, len(f.prompts))
	for _, prompt := range f.prompts {
		ids = append(ids, prompt.ID)
	}
	sort.Strings(ids)
	return ids
}
