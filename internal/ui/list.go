package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Item kinds.
const (
	KindStation = iota
	KindFavourite
	KindCategory
	KindPodcastFeed
	KindPodcastEpisode
	KindPreviewInfo
	KindPreviewMeta
)

// ListItem is a single row in a selectable list.
type ListItem struct {
	Name     string
	Codec    string
	Bitrate  int
	Country  string
	IsHeader bool
	Kind     int

	// Podcast-specific fields.
	Duration string
	Date     string
	Status   string // e.g. "●", "○", "✓"
	FeedURL  string // for locating the episode across subscriptions
	URL      string // episode media URL
}

// List is a simple selectable list that renders columns.
type List struct {
	Items  []ListItem
	Cursor int
}

// clampCursor ensures the cursor points inside the current item slice.
func (l *List) clampCursor() {
	if len(l.Items) == 0 {
		l.Cursor = 0
		return
	}
	if l.Cursor >= len(l.Items) {
		l.Cursor = len(l.Items) - 1
	}
	if l.Cursor < 0 {
		l.Cursor = 0
	}
}

// selectable reports whether the item at idx can be the active cursor position.
func (l *List) selectable(idx int) bool {
	if idx < 0 || idx >= len(l.Items) {
		return false
	}
	item := l.Items[idx]
	return !item.IsHeader && item.Kind != KindPreviewInfo && item.Kind != KindPreviewMeta
}

// SelectFirst moves the cursor to the first selectable item.
func (l *List) SelectFirst() {
	for i := range l.Items {
		if l.selectable(i) {
			l.Cursor = i
			return
		}
	}
	l.Cursor = 0
}

// Up moves the cursor up, skipping non-selectable items.
func (l *List) Up() {
	l.clampCursor()
	for l.Cursor > 0 {
		l.Cursor--
		if l.selectable(l.Cursor) {
			return
		}
	}
}

// Down moves the cursor down, skipping non-selectable items.
func (l *List) Down() {
	l.clampCursor()
	for l.Cursor < len(l.Items)-1 {
		l.Cursor++
		if l.selectable(l.Cursor) {
			return
		}
	}
}

// Selected returns the currently selected item, or nil.
func (l *List) Selected() *ListItem {
	if l.Cursor < 0 || l.Cursor >= len(l.Items) {
		return nil
	}
	return &l.Items[l.Cursor]
}

// View renders the list within the given dimensions.
func (l *List) View(width, height int, focused bool) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	l.clampCursor()

	var b strings.Builder
	visible := l.Items
	start := 0
	// Keep the cursor visible by scrolling.
	if l.Cursor >= height {
		start = l.Cursor - height + 1
	}
	if start+height > len(visible) {
		start = len(visible) - height
		if start < 0 {
			start = 0
		}
	}
	end := start + height
	if end > len(visible) {
		end = len(visible)
	}

	for i := start; i < end; i++ {
		item := visible[i]
		line := l.renderItem(item, width, focused && i == l.Cursor)
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (l *List) renderItem(item ListItem, width int, selected bool) string {
	if item.IsHeader {
		text := truncate(item.Name, width)
		return CategoryStyle.Render(text)
	}

	name := item.Name

	switch item.Kind {
	case KindPreviewInfo:
		return truncate(name, width)
	case KindPreviewMeta:
		return MutedStyle.Render(truncate(name, width))
	case KindPodcastEpisode:
		return l.renderPodcastEpisode(item, name, width, selected)
	case KindPodcastFeed:
		return l.renderPodcastFeed(item, name, width, selected)
	default:
		return l.renderStation(item, name, width, selected)
	}
}

func (l *List) renderStation(item ListItem, name string, width int, selected bool) string {
	codec := item.Codec
	if codec == "" {
		codec = "?"
	}
	bitrate := "?"
	if item.Bitrate > 0 {
		bitrate = fmt.Sprintf("%dk", item.Bitrate)
	}
	country := item.Country
	if country == "" {
		country = "?"
	}

	codecW := 8
	bitrateW := 8
	countryW := 6
	nameW := width - codecW - bitrateW - countryW - 3
	if nameW < 10 {
		nameW = 10
	}

	nameCol := padRight(truncate(name, nameW), nameW)
	codecCol := padRight(truncate(codec, codecW), codecW)
	bitrateCol := padRight(truncate(bitrate, bitrateW), bitrateW)
	countryCol := padRight(truncate(country, countryW), countryW)

	line := fmt.Sprintf("%s %s %s %s", nameCol, codecCol, bitrateCol, countryCol)
	line = truncate(line, width)

	if selected {
		return SelectedItemStyle.Render(line)
	}
	return line
}

func (l *List) renderPodcastFeed(item ListItem, name string, width int, selected bool) string {
	nameW := width - 2
	if nameW < 10 {
		nameW = 10
	}
	line := truncate(name, nameW)
	if selected {
		return SelectedItemStyle.Render(line)
	}
	return line
}

func (l *List) renderPodcastEpisode(item ListItem, name string, width int, selected bool) string {
	duration := item.Duration
	if duration == "" {
		duration = "?"
	}
	date := item.Date
	if date == "" {
		date = "?"
	}
	status := item.Status
	if status == "" {
		status = " "
	}

	durationW := 8
	dateW := 12
	statusW := 4
	nameW := width - durationW - dateW - statusW - 3
	if nameW < 10 {
		nameW = 10
	}

	nameCol := padRight(truncate(name, nameW), nameW)
	durCol := padRight(truncate(duration, durationW), durationW)
	dateCol := padRight(truncate(date, dateW), dateW)
	statusCol := padRight(truncate(status, statusW), statusW)

	line := fmt.Sprintf("%s %s %s %s", nameCol, durCol, dateCol, statusCol)
	line = truncate(line, width)

	if selected {
		return SelectedItemStyle.Render(line)
	}
	return line
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	w := runewidth.StringWidth(s)
	if w <= max {
		return s
	}
	return runewidth.Truncate(s, max, "")
}

func padRight(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
