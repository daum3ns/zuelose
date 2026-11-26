package ui

import "github.com/charmbracelet/lipgloss"

// HelpText returns the keyboard shortcut overlay.
func HelpText() string {
	text := `
Navigation
  ↑ / ↓        Move up / down in the focused pane
  Tab          Next pane
  Shift+Tab    Previous pane
  ← / →        Previous / next pane (when not typing)
  Enter        Activate selected item / play
  Esc          Back / cancel

Playback
  Space        Pause / resume (podcasts)
  Space        Connect / disconnect (radio streams)
  Shift+← / →  Seek ±10 s (podcasts)
  Shift+↑ / ↓  Seek ±60 s (podcasts)
  + / -        Volume up / down

Radio
  /            Focus search input
  F            Add station to favourites
  R            Refresh station list

Radio Favourites
  F            Add station to favourites / edit category
  D            Delete favourite
  C            Create category

Podcast Search
  /            Focus search input
  Enter        Preview episodes
  A            Subscribe to selected podcast

Podcast Subscriptions
  A            Add podcast feed by URL
  D            Delete subscription
  Enter        Open subscription episodes

Podcast Episodes
  Enter        Play episode (prompts if resumable)
  R            Resume episode from saved position

Other
  ?            Toggle this help
  Q            Quit
`
	box := lipgloss.NewStyle().
		Background(ColorSurface()).
		Foreground(ColorText()).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary())
	return box.Render(text)
}
