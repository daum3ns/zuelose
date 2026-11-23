package store

import (
	"os"
	"path/filepath"

	"github.com/daum3ns/zuelose/internal/podcast"
	"gopkg.in/yaml.v3"
)

// PodcastsPath returns the path to the podcast subscriptions file.
func PodcastsPath() string {
	return filepath.Join(ConfigDir(), "podcasts.yaml")
}

// LoadPodcasts loads podcast subscriptions and episode state from disk.
func LoadPodcasts() ([]podcast.Feed, error) {
	var feeds []podcast.Feed
	data, err := os.ReadFile(PodcastsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return feeds, nil
		}
		return feeds, err
	}
	if err := yaml.Unmarshal(data, &feeds); err != nil {
		return feeds, err
	}
	return feeds, nil
}

// SavePodcasts writes podcast subscriptions and episode state to disk.
func SavePodcasts(feeds []podcast.Feed) error {
	if err := ensureDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(feeds)
	if err != nil {
		return err
	}
	return os.WriteFile(PodcastsPath(), data, 0o644)
}

// FindEpisode locates a specific episode by feed URL and episode URL.
func FindEpisode(feeds []podcast.Feed, feedURL, episodeURL string) (*podcast.Feed, *podcast.Episode) {
	for i := range feeds {
		if feeds[i].URL != feedURL {
			continue
		}
		for j := range feeds[i].Episodes {
			if feeds[i].Episodes[j].URL == episodeURL {
				return &feeds[i], &feeds[i].Episodes[j]
			}
		}
	}
	return nil, nil
}
