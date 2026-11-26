package ui

// HelpText returns the complete keyboard reference shown in the scrollable help overlay.
func HelpText() string {
	return `Global
  ↑ / ↓          Move in the focused pane
  Tab            Next pane
  Shift+Tab      Previous pane
  ← / →          Previous / next pane (when not typing)
  Enter          Activate selected item
  Esc            Back / cancel / leave input
  ?              Toggle help
  q / Ctrl+c     Quit
  ● / ○          Focused / inactive pane

  Letter shortcuts are case-insensitive.

Playback
  Space          Pause / resume podcast
  Space          Connect / disconnect radio stream
  Shift+← / →    Seek ±10 s (podcasts)
  Shift+↑ / ↓    Seek ±60 s (podcasts)
  + / -          Volume up / down

Radio Browse
  /              Focus search input
  Enter          Play selected station
  f              Add station to favourites
  r              Search / refresh results
  Esc            Focus sidebar

Radio Favourites
  Enter          Play selected favourite
  f              Edit favourite category
  d              Remove favourite
  c              Create category
  Esc            Focus sidebar

Podcast Search
  /              Focus search input
  Enter          Preview selected podcast
  a              Subscribe to selected podcast
  r              Search / refresh results
  Esc            Focus sidebar

Podcast Preview
  Enter          Play selected episode
  a              Subscribe to previewed podcast
  /              Back to podcast search
  Esc            Back to podcast search

Podcast Subscriptions
  Enter          Open subscription episodes
  a              Add podcast feed by URL
  d              Unsubscribe
  r              Refresh subscriptions
  Esc            Focus sidebar

Podcast Episodes
  Enter          Play episode (prompts if resumable)
  r              Resume selected episode
  Esc            Back to subscriptions

New Episodes
  Enter          Play selected episode
  r              Refresh subscriptions
  Esc            Focus sidebar

Continue Listening
  Enter          Resume selected episode
  r              Refresh subscriptions
  Esc            Focus sidebar

Settings
  Enter / e      Edit or cycle setting
  Esc            Focus sidebar

Dialogs
  Enter          Confirm / save / fetch
  Esc            Cancel
  Tab            Complete category (save favourite)
  r / s          Resume / restart (resume prompt)

Help navigation
  ↑ / ↓ or j/k   Scroll one line
  PgUp / PgDn    Scroll one page
  Home / End     Jump to top / bottom
  Esc / ? / q    Close help
`
}
