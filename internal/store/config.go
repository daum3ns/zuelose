package store

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds user-level settings.
type Config struct {
	Volume                 int    `yaml:"volume"`
	MpvPath                string `yaml:"mpv_path"`
	APIHost                string `yaml:"api_host"`
	PodcastRefreshInterval int    `yaml:"podcast_refresh_interval"` // minutes
	PodcastBrowser         string `yaml:"podcast_browser"`
	Theme                  string `yaml:"theme"`
	ThemeMode              string `yaml:"theme_mode"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Volume:                 75,
		MpvPath:                "mpv",
		APIHost:                "all.api.radio-browser.info",
		PodcastRefreshInterval: 60,
		PodcastBrowser:         "itunes",
		Theme:                  "tokyo-night",
		ThemeMode:              "dark",
	}
}

// ConfigDir returns the directory used for zuelose configuration.
func ConfigDir() string {
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "zuelose")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "zuelose")
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// FavouritesPath returns the full path to the favourites file.
func FavouritesPath() string {
	return filepath.Join(ConfigDir(), "favourites.yaml")
}

// ensureDir makes sure the configuration directory exists.
func ensureDir() error {
	dir := ConfigDir()
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// LoadConfig loads the configuration from disk, returning defaults if missing.
func LoadConfig() (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// SaveConfig writes the configuration to disk.
func SaveConfig(cfg Config) error {
	if err := ensureDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0o644)
}
