package ui

import "github.com/charmbracelet/lipgloss"

// Theme colours.
var (
	ColorPrimary   = lipgloss.Color("#7AA2F7")
	ColorSecondary = lipgloss.Color("#BB9AF7")
	ColorSuccess   = lipgloss.Color("#9ECE6A")
	ColorWarn      = lipgloss.Color("#E0AF68")
	ColorError     = lipgloss.Color("#F7768E")
	ColorMuted     = lipgloss.Color("#565F89")
	ColorText      = lipgloss.Color("#C0CAF5")
	ColorBg        = lipgloss.Color("#1A1B26")
	ColorSurface   = lipgloss.Color("#24283B")
)

// Common styles.
var (
	AppStyle = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText)

	HeaderStyle = lipgloss.NewStyle().
			Background(ColorSurface).
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary)

	SidebarStyle = lipgloss.NewStyle().
				Background(ColorBg).
				Foreground(ColorText).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary)

	SidebarSelectedStyle = lipgloss.NewStyle().
						Background(ColorPrimary).
						Foreground(ColorBg).
						Bold(true).
						Padding(0, 1)

	MainStyle = lipgloss.NewStyle().
			Background(ColorBg).
			Foreground(ColorText).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary)

	SelectedItemStyle = lipgloss.NewStyle().
					Background(ColorSecondary).
					Foreground(ColorBg).
					Bold(true)

	CategoryStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
