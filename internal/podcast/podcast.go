package podcast

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// Feed represents a podcast subscription.
type Feed struct {
	Title       string    `yaml:"title"`
	URL         string    `yaml:"url"`
	Description string    `yaml:"description,omitempty"`
	ImageURL    string    `yaml:"image_url,omitempty"`
	Categories  []string  `yaml:"categories,omitempty"`
	LastFetched time.Time `yaml:"last_fetched,omitempty"`
	Episodes    []Episode `yaml:"episodes"`
}

// Episode represents a single podcast episode.
type Episode struct {
	Title       string    `yaml:"title"`
	URL         string    `yaml:"url"`
	Description string    `yaml:"description,omitempty"`
	PubDate     time.Time `yaml:"pub_date"`
	Duration    int       `yaml:"duration"` // seconds, 0 if unknown
	Position    int       `yaml:"position"` // last playback position in seconds
	Played      bool      `yaml:"played"`
	Completed   bool      `yaml:"completed"`
	IsNew       bool      `yaml:"is_new,omitempty"`
}

// SearchResult is returned by a podcast directory search.
type SearchResult struct {
	Title    string
	URL      string
	ImageURL string
	Artist   string
}

// SearchITunes searches the iTunes podcast directory for the given term.
func SearchITunes(ctx context.Context, term string) ([]SearchResult, error) {
	u := "https://itunes.apple.com/search?media=podcast&limit=50&term=" + url.QueryEscape(term)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itunes returned %s", resp.Status)
	}

	var payload struct {
		Results []struct {
			CollectionName string `json:"collectionName"`
			FeedURL        string `json:"feedUrl"`
			ArtworkURL600  string `json:"artworkUrl600"`
			ArtistName     string `json:"artistName"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(payload.Results))
	for _, r := range payload.Results {
		if r.FeedURL == "" {
			continue
		}
		results = append(results, SearchResult{
			Title:    r.CollectionName,
			URL:      r.FeedURL,
			ImageURL: r.ArtworkURL600,
			Artist:   r.ArtistName,
		})
	}
	return results, nil
}

// ParseFeed fetches and parses an RSS/Atom podcast feed.
func ParseFeed(ctx context.Context, feedURL string) (*Feed, error) {
	fp := gofeed.NewParser()
	fp.Client = &http.Client{Timeout: 20 * time.Second}
	parsed, err := fp.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, err
	}

	feed := &Feed{
		Title:       parsed.Title,
		URL:         feedURL,
		Description: stripHTML(parsed.Description),
		ImageURL:    imageFromFeed(parsed),
		Categories:  parsed.Categories,
	}

	for _, item := range parsed.Items {
		if item == nil {
			continue
		}
		audioURL := audioURLFromItem(item)
		if audioURL == "" {
			continue
		}
		feed.Episodes = append(feed.Episodes, Episode{
			Title:       item.Title,
			URL:         audioURL,
			Description: stripHTML(item.Description),
			PubDate:     publishedTime(item),
			Duration:    durationFromItem(item),
		})
	}
	return feed, nil
}

func audioURLFromItem(item *gofeed.Item) string {
	for _, e := range item.Enclosures {
		if e != nil && strings.HasPrefix(e.Type, "audio/") && e.URL != "" {
			return e.URL
		}
	}
	// Fallback: return first enclosure URL regardless of type.
	if len(item.Enclosures) > 0 && item.Enclosures[0] != nil {
		return item.Enclosures[0].URL
	}
	return ""
}

func imageFromFeed(feed *gofeed.Feed) string {
	if feed.Image != nil && feed.Image.URL != "" {
		return feed.Image.URL
	}
	for _, ext := range feed.Extensions {
		if ext == nil {
			continue
		}
		if itunes, ok := ext["image"]; ok {
			if len(itunes) > 0 && itunes[0].Attrs != nil {
				if href := itunes[0].Attrs["href"]; href != "" {
					return href
				}
			}
		}
	}
	return ""
}

func durationFromItem(item *gofeed.Item) int {
	if item.ITunesExt != nil && item.ITunesExt.Duration != "" {
		return parseDuration(item.ITunesExt.Duration)
	}
	return 0
}

func parseDuration(s string) int {
	parts := strings.Split(s, ":")
	if len(parts) == 0 {
		return 0
	}
	// Try HH:MM:SS, MM:SS, or seconds.
	var total int
	mult := 1
	for i := len(parts) - 1; i >= 0; i-- {
		var v int
		fmt.Sscanf(parts[i], "%d", &v)
		total += v * mult
		mult *= 60
	}
	return total
}

func publishedTime(item *gofeed.Item) time.Time {
	if item.PublishedParsed != nil {
		return *item.PublishedParsed
	}
	return time.Time{}
}

func stripHTML(s string) string {
	// Very basic HTML stripping for readability.
	out := strings.ReplaceAll(s, "<p>", "\n")
	out = strings.ReplaceAll(out, "</p>", "")
	out = strings.ReplaceAll(out, "<br>", "\n")
	out = strings.ReplaceAll(out, "<br/>", "\n")
	return strings.TrimSpace(out)
}
