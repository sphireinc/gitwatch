// Package theme defines semantic terminal styles and colorless fallbacks.
package theme

import (
	"os"
	"strings"

	"charm.land/lipgloss/v2"
)

type Name string

const (
	Auto         Name = "auto"
	Dark         Name = "dark"
	Light        Name = "light"
	HighContrast Name = "high-contrast"
)

type Roles struct {
	Clean, Modified, Staged, Untracked, Conflict, Deleted, Muted, Selection, Success, Warning, Error, Header, Border lipgloss.Style
	Colorless                                                                                                        bool
}

func New(name Name, noColor bool) Roles {
	if os.Getenv("NO_COLOR") != "" {
		noColor = true
	}
	dark := name != Light
	if name == Auto {
		dark = true
	}
	roles := Roles{Colorless: noColor}
	style := func(fg string) lipgloss.Style {
		if noColor {
			return lipgloss.NewStyle()
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color(fg))
	}
	selection := func(fg, bg string) lipgloss.Style {
		if noColor {
			return lipgloss.NewStyle()
		}
		return style(fg).Background(lipgloss.Color(bg))
	}
	if dark {
		roles.Clean = style("#7ee787")
		roles.Modified = style("#f2cc60")
		roles.Staged = style("#58a6ff")
		roles.Untracked = style("#d2a8ff")
		roles.Conflict = style("#ff7b72")
		roles.Deleted = style("#ff7b72")
		roles.Muted = style("#8b949e")
		roles.Selection = selection("#ffffff", "#30363d")
		roles.Success = style("#7ee787")
		roles.Warning = style("#f2cc60")
		roles.Error = style("#ff7b72")
		roles.Header = style("#79c0ff")
		roles.Border = style("#484f58")
	} else {
		roles.Clean = style("#1a7f37")
		roles.Modified = style("#9a6700")
		roles.Staged = style("#0969da")
		roles.Untracked = style("#8250df")
		roles.Conflict = style("#cf222e")
		roles.Deleted = style("#cf222e")
		roles.Muted = style("#57606a")
		roles.Selection = selection("#24292f", "#ddf4ff")
		roles.Success = style("#1a7f37")
		roles.Warning = style("#9a6700")
		roles.Error = style("#cf222e")
		roles.Header = style("#0550ae")
		roles.Border = style("#8c959f")
	}
	if name == HighContrast {
		roles.Selection = lipgloss.NewStyle().Bold(true).Underline(true)
		roles.Colorless = noColor
	}
	return roles
}

func Symbol(status string) string {
	switch strings.ToLower(status) {
	case "staged":
		return "S"
	case "modified", "staged + modified":
		return "M"
	case "untracked":
		return "?"
	case "conflict":
		return "!"
	case "deleted":
		return "D"
	case "renamed":
		return "R"
	default:
		return "·"
	}
}
