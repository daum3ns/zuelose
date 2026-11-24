package store

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Station represents a radio station from any source.
type Station struct {
	Name        string `yaml:"name"`
	URL         string `yaml:"url"`
	URLResolved string `yaml:"url_resolved,omitempty"`
	Favicon     string `yaml:"favicon,omitempty"`
	Codec       string `yaml:"codec"`
	Bitrate     int    `yaml:"bitrate"`
	Country     string `yaml:"country"`
	CountryCode string `yaml:"countrycode,omitempty"`
	Tags        string `yaml:"tags,omitempty"`
	UUID        string `yaml:"uuid,omitempty"`
}

// PlayURL returns the best available stream URL.
func (s Station) PlayURL() string {
	if s.URLResolved != "" {
		return s.URLResolved
	}
	return s.URL
}

// FavStation is a saved station.
type FavStation struct {
	Station
	Category string `yaml:"-"`
}

// Category groups favourite stations.
type Category struct {
	Name     string       `yaml:"name"`
	Stations []FavStation `yaml:"stations"`
}

// Favourites holds all saved categories.
type Favourites struct {
	Categories []Category `yaml:"categories"`
}

// LoadFavourites loads favourites from disk.
func LoadFavourites() (Favourites, error) {
	var favs Favourites
	data, err := os.ReadFile(FavouritesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return favs, nil
		}
		return favs, err
	}
	if err := yaml.Unmarshal(data, &favs); err != nil {
		return favs, err
	}
	return favs, nil
}

// SaveFavourites writes favourites to disk.
func SaveFavourites(favs Favourites) error {
	if err := ensureDir(); err != nil {
		return err
	}
	data, err := yaml.Marshal(&favs)
	if err != nil {
		return err
	}
	return os.WriteFile(FavouritesPath(), data, 0o644)
}

// CategoryNames returns a sorted slice of category names.
func (f *Favourites) CategoryNames() []string {
	names := make([]string, len(f.Categories))
	for i, c := range f.Categories {
		names[i] = c.Name
	}
	return names
}

// Add saves a station under the given category.
func (f *Favourites) Add(st Station, category string) error {
	if category == "" {
		return errors.New("category name is required")
	}
	catIdx := -1
	for i, c := range f.Categories {
		if c.Name == category {
			catIdx = i
			break
		}
	}
	if catIdx == -1 {
		f.Categories = append(f.Categories, Category{Name: category})
		catIdx = len(f.Categories) - 1
	}
	cat := &f.Categories[catIdx]
	for _, s := range cat.Stations {
		if s.Name == st.Name {
			return fmt.Errorf("station %q already exists in category %q", st.Name, category)
		}
	}
	cat.Stations = append(cat.Stations, FavStation{
		Station:  st,
		Category: category,
	})
	return nil
}

// Remove deletes a favourite by category and station name. Empty categories are
// removed so that the favourites list does not accumulate dead groups.
func (f *Favourites) Remove(category, name string) {
	for i := range f.Categories {
		if f.Categories[i].Name != category {
			continue
		}
		filtered := f.Categories[i].Stations[:0]
		for _, s := range f.Categories[i].Stations {
			if s.Name != name {
				filtered = append(filtered, s)
			}
		}
		f.Categories[i].Stations = filtered
		if len(f.Categories[i].Stations) == 0 {
			f.Categories = append(f.Categories[:i], f.Categories[i+1:]...)
		}
		return
	}
}

// RenameCategory renames an existing category.
func (f *Favourites) RenameCategory(oldName, newName string) error {
	if newName == "" {
		return errors.New("category name cannot be empty")
	}
	for _, c := range f.Categories {
		if c.Name == newName && c.Name != oldName {
			return fmt.Errorf("category %q already exists", newName)
		}
	}
	for i := range f.Categories {
		if f.Categories[i].Name == oldName {
			f.Categories[i].Name = newName
			for j := range f.Categories[i].Stations {
				f.Categories[i].Stations[j].Category = newName
			}
			return nil
		}
	}
	return fmt.Errorf("category %q not found", oldName)
}

// CreateCategory adds an empty category if it does not exist.
func (f *Favourites) CreateCategory(name string) error {
	if name == "" {
		return errors.New("category name cannot be empty")
	}
	for _, c := range f.Categories {
		if c.Name == name {
			return fmt.Errorf("category %q already exists", name)
		}
	}
	f.Categories = append(f.Categories, Category{Name: name})
	return nil
}
