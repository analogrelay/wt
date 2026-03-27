package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Purple    = lipgloss.Color("99")
	Blue      = lipgloss.Color("39")
	Green     = lipgloss.Color("78")
	Yellow    = lipgloss.Color("220")
	Red       = lipgloss.Color("196")
	Gray      = lipgloss.Color("245")
	DarkGray  = lipgloss.Color("238")
	White     = lipgloss.Color("255")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Purple)

	PromptStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	NormalStyle = lipgloss.NewStyle().
			Foreground(White)

	DimStyle = lipgloss.NewStyle().
			Foreground(Gray)

	AnnotationStyle = lipgloss.NewStyle().
			Foreground(Gray)

	SessionStyle = lipgloss.NewStyle().
			Foreground(Green)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Red)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Yellow)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(White)

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(DarkGray)

	CursorStyle = lipgloss.NewStyle().
			Foreground(Blue)

	FilterStyle = lipgloss.NewStyle().
			Foreground(Yellow)
)

// Status indicators matching the shell version.
const (
	IndicatorClean   = "✓"
	IndicatorDirty   = "✗"
	IndicatorSession = "●"
	IndicatorNone    = "—"
)
