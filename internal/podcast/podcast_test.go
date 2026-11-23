package podcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mmcdole/gofeed"
)

const testRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd" version="2.0">
  <channel>
    <title>Test Podcast</title>
    <description><p>A test podcast.</p></description>
    <itunes:image href="http://example.com/image.jpg"/>
    <item>
      <title>Episode 1</title>
      <pubDate>Mon, 15 Jan 2024 00:00:00 GMT</pubDate>
      <itunes:duration>1:05:30</itunes:duration>
      <enclosure url="http://example.com/ep1.mp3" type="audio/mpeg"/>
    </item>
    <item>
      <title>Episode 2</title>
      <pubDate>Tue, 16 Jan 2024 00:00:00 GMT</pubDate>
      <enclosure url="http://example.com/ep2.mp3" type="audio/mpeg"/>
    </item>
  </channel>
</rss>`

func TestParseFeed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.Write([]byte(testRSS))
	}))
	defer server.Close()

	feed, err := ParseFeed(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if feed.Title != "Test Podcast" {
		t.Fatalf("expected title Test Podcast, got %q", feed.Title)
	}
	if feed.ImageURL != "http://example.com/image.jpg" {
		t.Fatalf("expected image URL, got %q", feed.ImageURL)
	}
	if len(feed.Episodes) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(feed.Episodes))
	}
	if feed.Episodes[0].Title != "Episode 1" {
		t.Fatalf("unexpected episode title: %q", feed.Episodes[0].Title)
	}
	if feed.Episodes[0].Duration != 3930 {
		t.Fatalf("expected duration 3930, got %d", feed.Episodes[0].Duration)
	}
	if feed.Episodes[0].URL != "http://example.com/ep1.mp3" {
		t.Fatalf("unexpected episode URL: %q", feed.Episodes[0].URL)
	}
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"45", 45},
		{"5:30", 330},
		{"1:05:30", 3930},
		{"bad", 0},
	}
	for _, c := range cases {
		if got := parseDuration(c.in); got != c.want {
			t.Errorf("parseDuration(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestAudioURLFromItem(t *testing.T) {
	item := &gofeed.Item{
		Enclosures: []*gofeed.Enclosure{
			{URL: "http://video.mp4", Type: "video/mp4"},
			{URL: "http://audio.mp3", Type: "audio/mpeg"},
		},
	}
	if got := audioURLFromItem(item); got != "http://audio.mp3" {
		t.Fatalf("expected audio URL, got %q", got)
	}
}

func TestStripHTML(t *testing.T) {
	got := stripHTML("<p>Hello</p><br>World")
	want := "Hello\nWorld"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

