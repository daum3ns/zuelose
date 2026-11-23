package ui

import (
	"strings"
	"testing"
)

func TestListNavigation(t *testing.T) {
	list := List{
		Items: []ListItem{
			{Name: "Header", IsHeader: true},
			{Name: "One"},
			{Name: "Two"},
			{Name: "Subheader", IsHeader: true},
			{Name: "Three"},
		},
	}

	list.Down()
	if got := list.Selected().Name; got != "One" {
		t.Fatalf("expected One, got %q", got)
	}

	list.Down()
	if got := list.Selected().Name; got != "Two" {
		t.Fatalf("expected Two, got %q", got)
	}

	// Should skip Subheader.
	list.Down()
	if got := list.Selected().Name; got != "Three" {
		t.Fatalf("expected Three, got %q", got)
	}

	// Should stay at last item.
	list.Down()
	if got := list.Selected().Name; got != "Three" {
		t.Fatalf("expected Three, got %q", got)
	}

	// Should skip Subheader going up.
	list.Up()
	if got := list.Selected().Name; got != "Two" {
		t.Fatalf("expected Two, got %q", got)
	}
}

func TestListEmpty(t *testing.T) {
	list := List{Items: nil}
	if list.Selected() != nil {
		t.Fatal("expected nil selection for empty list")
	}
	list.Down()
	list.Up()
	if list.Cursor != 0 {
		t.Fatalf("expected cursor 0, got %d", list.Cursor)
	}
}

func TestListViewScrolls(t *testing.T) {
	items := make([]ListItem, 10)
	for i := range items {
		items[i] = ListItem{Name: strings.Repeat("x", i+1)}
	}
	list := List{Items: items, Cursor: 9}

	view := list.View(20, 3, false)
	lines := strings.Split(view, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[2], "xxxxxxxxxx") {
		t.Fatalf("expected last item visible, got %q", lines[2])
	}
}

func TestListViewEmpty(t *testing.T) {
	list := List{Items: nil}
	if got := list.View(10, 5, false); got != "" {
		t.Fatalf("expected empty view, got %q", got)
	}
}

func TestRenderStationColumns(t *testing.T) {
	list := List{Items: []ListItem{{
		Name:    "Station",
		Codec:   "MP3",
		Bitrate: 128,
		Country: "CH",
	}}}
	view := list.View(40, 1, false)
	if !strings.Contains(view, "Station") {
		t.Fatalf("expected name in view, got %q", view)
	}
	if !strings.Contains(view, "MP3") {
		t.Fatalf("expected codec in view, got %q", view)
	}
	if !strings.Contains(view, "128k") {
		t.Fatalf("expected bitrate in view, got %q", view)
	}
	if !strings.Contains(view, "CH") {
		t.Fatalf("expected country in view, got %q", view)
	}
}

func TestRenderPodcastEpisodeColumns(t *testing.T) {
	list := List{Items: []ListItem{{
		Name:     "Episode",
		Duration: "12:34",
		Date:     "2024-01-15",
		Status:   "●",
		Kind:     KindPodcastEpisode,
	}}}
	view := list.View(50, 1, false)
	if !strings.Contains(view, "Episode") {
		t.Fatalf("expected name in view, got %q", view)
	}
	if !strings.Contains(view, "12:34") {
		t.Fatalf("expected duration in view, got %q", view)
	}
	if !strings.Contains(view, "2024-01-15") {
		t.Fatalf("expected date in view, got %q", view)
	}
}
