package ui

import "github.com/charmbracelet/lipgloss"

// Common styles. These are recreated whenever SetTheme is called.
var (
	AppStyle = lipgloss.NewStyle().
		Background(ColorBg()).
		Foreground(ColorText())

	HeaderStyle = lipgloss.NewStyle().
			Background(ColorSurface()).
			Foreground(ColorPrimary()).
			Bold(true).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Background(ColorBg()).
			Foreground(ColorText()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary())

	SidebarStyle = lipgloss.NewStyle().
			Background(ColorBg()).
			Foreground(ColorText()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary())

	SidebarSelectedStyle = lipgloss.NewStyle().
				Background(ColorPrimary()).
				Foreground(ColorBg()).
				Bold(true).
				Padding(0, 1)

	MainStyle = lipgloss.NewStyle().
			Background(ColorBg()).
			Foreground(ColorText()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary())

	SelectedItemStyle = lipgloss.NewStyle().
				Background(ColorSecondary()).
				Foreground(ColorBg()).
				Bold(true)

	CategoryStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess()).
				Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted())

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError()).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted())
)

func recreateStyles() {
	AppStyle = lipgloss.NewStyle().
		Background(ColorBg()).
		Foreground(ColorText())

	HeaderStyle = lipgloss.NewStyle().
			Background(ColorSurface()).
			Foreground(ColorPrimary()).
			Bold(true).
			Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Background(ColorBg()).
			Foreground(ColorText()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary())

	SidebarStyle = lipgloss.NewStyle().
			Background(ColorBg()).
			Foreground(ColorText()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary())

	SidebarSelectedStyle = lipgloss.NewStyle().
				Background(ColorPrimary()).
				Foreground(ColorBg()).
				Bold(true).
				Padding(0, 1)

	MainStyle = lipgloss.NewStyle().
			Background(ColorBg()).
			Foreground(ColorText()).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary())

	SelectedItemStyle = lipgloss.NewStyle().
				Background(ColorSecondary()).
				Foreground(ColorBg()).
				Bold(true)

	CategoryStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess()).
				Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted())

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError()).
			Bold(true)

	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted())
}
