package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daum3ns/zuelose/internal/podcast"
)

func setConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
}

func TestConfigLoadSave(t *testing.T) {
	setConfigDir(t)

	cfg := DefaultConfig()
	cfg.Volume = 42
	cfg.PodcastRefreshInterval = 30
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Volume != 42 {
		t.Fatalf("expected volume 42, got %d", loaded.Volume)
	}
	if loaded.PodcastRefreshInterval != 30 {
		t.Fatalf("expected interval 30, got %d", loaded.PodcastRefreshInterval)
	}
	if loaded.PodcastBrowser != "itunes" {
		t.Fatalf("expected browser itunes, got %q", loaded.PodcastBrowser)
	}
}

func TestConfigDefaultsOnMissingFile(t *testing.T) {
	setConfigDir(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Volume != 75 {
		t.Fatalf("expected default volume 75, got %d", cfg.Volume)
	}
}

func TestFavouritesLoadSave(t *testing.T) {
	setConfigDir(t)

	favs := Favourites{Categories: []Category{{
		Name: "Test",
		Stations: []FavStation{{
			Station: Station{Name: "Radio 1", URL: "http://example.com/stream"},
		}},
	}}}
	if err := SaveFavourites(favs); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadFavourites()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Categories) != 1 || loaded.Categories[0].Name != "Test" {
		t.Fatalf("unexpected categories: %+v", loaded.Categories)
	}
}

func TestFavouritesAddAndRemove(t *testing.T) {
	favs := Favourites{}
	if err := favs.Add(Station{Name: "S1", URL: "http://s1"}, "Cat"); err != nil {
		t.Fatal(err)
	}
	if err := favs.Add(Station{Name: "S1", URL: "http://s1"}, "Cat"); err == nil {
		t.Fatal("expected duplicate error")
	}
	favs.Remove("Cat", "S1")
	if len(favs.Categories) != 0 {
		t.Fatalf("expected empty category to be removed, got %+v", favs.Categories)
	}
}

func TestFavouritesRemoveOnlyEmptyCategory(t *testing.T) {
	favs := Favourites{}
	if err := favs.Add(Station{Name: "S1", URL: "http://s1"}, "Cat"); err != nil {
		t.Fatal(err)
	}
	if err := favs.Add(Station{Name: "S2", URL: "http://s2"}, "Cat"); err != nil {
		t.Fatal(err)
	}
	if err := favs.Add(Station{Name: "S3", URL: "http://s3"}, "Other"); err != nil {
		t.Fatal(err)
	}
	favs.Remove("Cat", "S1")
	if len(favs.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(favs.Categories))
	}
	favs.Remove("Cat", "S2")
	if len(favs.Categories) != 1 || favs.Categories[0].Name != "Other" {
		t.Fatalf("expected only 'Other', got %+v", favs.Categories)
	}
}

func TestPodcastsLoadSave(t *testing.T) {
	setConfigDir(t)

	feeds := []podcast.Feed{{
		Title: "Test Feed",
		URL:   "http://feed",
		Episodes: []podcast.Episode{{
			Title: "Episode 1",
			URL:   "http://episode1",
		}},
	}}
	if err := SavePodcasts(feeds); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPodcasts()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Title != "Test Feed" {
		t.Fatalf("unexpected feeds: %+v", loaded)
	}
}

func TestFindEpisode(t *testing.T) {
	feeds := []podcast.Feed{{
		URL: "http://feed",
		Episodes: []podcast.Episode{
			{URL: "http://ep1"},
			{URL: "http://ep2"},
		},
	}}
	feed, ep := FindEpisode(feeds, "http://feed", "http://ep2")
	if feed == nil || ep == nil {
		t.Fatal("expected to find episode")
	}
	if ep.URL != "http://ep2" {
		t.Fatalf("expected ep2, got %q", ep.URL)
	}
	if _, found := FindEpisode(feeds, "http://missing", "http://ep1"); found != nil {
		t.Fatal("expected nil for missing feed")
	}
}

func TestConfigDir(t *testing.T) {
	setConfigDir(t)
	dir := ConfigDir()
	if dir == "" {
		t.Fatal("expected non-empty config dir")
	}
	if filepath.Base(dir) != "zuelose" {
		t.Fatalf("expected config dir named zuelose, got %q", dir)
	}
	// ConfigDir only returns the path; ensureDir creates it.
	if err := ensureDir(); err != nil {
		t.Fatalf("ensureDir failed: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("config dir should exist after ensureDir: %v", err)
	}
}
