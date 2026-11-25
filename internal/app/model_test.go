package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/daum3ns/zuelose/internal/player"
	"github.com/daum3ns/zuelose/internal/podcast"
	"github.com/daum3ns/zuelose/internal/store"
	"github.com/daum3ns/zuelose/internal/ui"
)

type fakePlayer struct {
	played      []player.Media
	paused      bool
	volume      int
	started     bool
	seeked      []float64
	toggleCnt   int
	currentMedia player.Media
	playing     bool
}

func (f *fakePlayer) Start() error        { f.started = true; return nil }
func (f *fakePlayer) Play(m player.Media) { f.played = append(f.played, m); f.currentMedia = m; f.playing = true }
func (f *fakePlayer) Stop()               { f.playing = false }
func (f *fakePlayer) Seek(seconds float64) { f.seeked = append(f.seeked, seconds) }
func (f *fakePlayer) VolumeUp()           { f.volume += 5 }
func (f *fakePlayer) VolumeDown()         { f.volume -= 5 }
func (f *fakePlayer) SetVolume(v int)     { f.volume = v }
func (f *fakePlayer) TogglePause() {
	f.toggleCnt++
	if f.currentMedia.URL == "" {
		return
	}
	if f.currentMedia.IsPodcast {
		f.paused = !f.paused
		return
	}
	if f.playing {
		f.playing = false
		return
	}
	f.played = append(f.played, f.currentMedia)
	f.playing = true
}

type fakeRadio struct {
	results []store.Station
}

func (f *fakeRadio) Search(ctx context.Context, name string, limit int) ([]store.Station, error) {
	return f.results, nil
}

func newTestModel(t *testing.T) (model, *fakePlayer, *fakeRadio) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	fp := &fakePlayer{}
	fr := &fakeRadio{}
	m := NewModel(store.DefaultConfig(), store.Favourites{}, fp, fr)
	m = updateModel(m, tea.WindowSizeMsg{Width: 80, Height: 24})
	return m, fp, fr
}

func updateModel(m model, msg tea.Msg) model {
	newM, _ := m.Update(msg)
	return newM.(model)
}

func activate(m model, v view) model {
	newM, _ := m.activateView(v)
	return newM.(model)
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "?"},
		{-1, "?"},
		{45, "0:45"},
		{125, "2:05"},
		{3661, "1:01:01"},
	}
	for _, c := range cases {
		if got := formatDuration(c.in); got != c.want {
			t.Errorf("formatDuration(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	if got := formatDate(time.Time{}); got != "?" {
		t.Errorf("expected ?, got %q", got)
	}
	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	if got := formatDate(d); got != "2024-01-15" {
		t.Errorf("expected 2024-01-15, got %q", got)
	}
}

func TestMergeEpisodesPreservesState(t *testing.T) {
	existing := []podcast.Episode{{
		URL:       "http://ep1",
		Position:  120,
		Played:    true,
		Completed: false,
		IsNew:     false,
	}}
	fetched := []podcast.Episode{{
		URL:      "http://ep1",
		Duration: 300,
	}, {
		URL:      "http://ep2",
		Duration: 200,
	}}

	merged := mergeEpisodes(fetched, existing)
	if len(merged) != 2 {
		t.Fatalf("expected 2 episodes, got %d", len(merged))
	}
	if merged[0].Position != 120 || !merged[0].Played || merged[0].Completed {
		t.Fatalf("state not preserved: %+v", merged[0])
	}
	if !merged[1].IsNew {
		t.Fatalf("new episode should be marked IsNew: %+v", merged[1])
	}
}

func TestBuildNewEpisodeItems(t *testing.T) {
	m := model{
		podcasts: []podcast.Feed{{
			Title: "Feed A",
			URL:   "http://feed-a",
			Episodes: []podcast.Episode{
				{Title: "New One", URL: "http://ep1", IsNew: true, PubDate: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
				{Title: "Old One", URL: "http://ep2", IsNew: false, PubDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}},
	}
	items := m.buildNewEpisodeItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 new item, got %d", len(items))
	}
	if items[0].Name != "Feed A — New One" {
		t.Fatalf("unexpected name: %q", items[0].Name)
	}
	if items[0].FeedURL != "http://feed-a" || items[0].URL != "http://ep1" {
		t.Fatalf("unexpected URLs: %+v", items[0])
	}
}

func TestBuildContinueListeningItems(t *testing.T) {
	m := model{
		podcasts: []podcast.Feed{{
			Title: "Feed A",
			URL:   "http://feed-a",
			Episodes: []podcast.Episode{
				{Title: "In Progress", URL: "http://ep1", Position: 120, Duration: 300},
				{Title: "Completed", URL: "http://ep2", Position: 300, Duration: 300, Completed: true},
				{Title: "Unplayed", URL: "http://ep3", Position: 0, Duration: 300},
			},
		}},
	}
	items := m.buildContinueListeningItems()
	if len(items) != 1 {
		t.Fatalf("expected 1 continue item, got %d", len(items))
	}
	if items[0].Name != "Feed A — In Progress" {
		t.Fatalf("unexpected name: %q", items[0].Name)
	}
}

func TestSettingsCount(t *testing.T) {
	m := model{}
	if m.settingsCount() != 7 {
		t.Fatalf("expected 7 settings, got %d", m.settingsCount())
	}
}

func TestThemeModeCycle(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = activate(m, viewSettings)
	m.focus = focusList
	m.settingsCursor = 5
	before := m.config.ThemeMode
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.config.ThemeMode == before {
		t.Fatal("expected theme mode to change")
	}
	if ui.CurrentTheme.Mode != m.config.ThemeMode {
		t.Fatalf("theme mode not applied: %s vs %s", ui.CurrentTheme.Mode, m.config.ThemeMode)
	}
	if m.editingSetting {
		t.Fatal("expected no editor popup for theme mode")
	}
}

func TestThemeCycleRespectsMode(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = activate(m, viewSettings)
	m.focus = focusList
	m.settingsCursor = 6
	before := m.config.Theme
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.config.Theme == before {
		t.Fatal("expected theme to change")
	}
	if ui.CurrentTheme.Name != m.config.Theme {
		t.Fatalf("theme not applied: %s vs %s", ui.CurrentTheme.Name, m.config.Theme)
	}
	if ui.CurrentTheme.Mode != m.config.ThemeMode {
		t.Fatalf("theme mode mismatch: %s vs %s", ui.CurrentTheme.Mode, m.config.ThemeMode)
	}
	if m.editingSetting {
		t.Fatal("expected no editor popup for theme")
	}
}

func TestPodcastBrowserCycleDirect(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = activate(m, viewSettings)
	m.focus = focusList
	m.settingsCursor = 4
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.editingSetting {
		t.Fatal("expected no editor popup for podcast browser")
	}
}

func TestTextSettingOpensEditor(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = activate(m, viewSettings)
	m.focus = focusList
	m.settingsCursor = 0
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if !m.editingSetting {
		t.Fatal("expected editor popup for text setting")
	}
}

func TestCurrentListCoversPodcastViews(t *testing.T) {
	m := model{}
	views := []view{viewRadioBrowse, viewRadioFavourites, viewPodcastSearch, viewPodcastSubscriptions, viewPodcastEpisodes, viewPodcastNewEpisodes, viewPodcastContinueListening}
	for _, v := range views {
		m.view = v
		l := m.currentList()
		if l == nil {
			t.Fatalf("currentList nil for view %s", v)
		}
	}
}

func TestStationPlayURL(t *testing.T) {
	st := store.Station{URL: "http://fallback", URLResolved: "http://resolved"}
	if got := st.PlayURL(); got != "http://resolved" {
		t.Fatalf("expected resolved URL, got %q", got)
	}
	st.URLResolved = ""
	if got := st.PlayURL(); got != "http://fallback" {
		t.Fatalf("expected fallback URL, got %q", got)
	}
}

func TestListItemKind(t *testing.T) {
	if ui.KindStation != 0 {
		t.Fatalf("unexpected KindStation value: %d", ui.KindStation)
	}
}

func TestRadioEnterPlaysStation(t *testing.T) {
	m, fp, fr := newTestModel(t)
	fr.results = []store.Station{{Name: "Radio One", URL: "http://one", Codec: "MP3", Bitrate: 128}}
	m = updateModel(m, searchMsg{stations: fr.results})

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})

	if len(fp.played) != 1 {
		t.Fatalf("expected 1 played media, got %d", len(fp.played))
	}
	if fp.played[0].Name != "Radio One" {
		t.Fatalf("expected Radio One, got %q", fp.played[0].Name)
	}
}

func TestSlashFocusesSearchInRadioBrowse(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.focus != focusSearch {
		t.Fatalf("expected focusSearch, got %d", m.focus)
	}
}

func TestGlobalVolumeKeys(t *testing.T) {
	m, fp, _ := newTestModel(t)
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})
	if fp.volume <= 0 {
		t.Fatalf("expected volume increased, got %d", fp.volume)
	}
}

func TestSettingsNavigation(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = activate(m, viewSettings)
	if m.view != viewSettings {
		t.Fatalf("expected settings view, got %s", m.view)
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyDown})
	if m.settingsCursor != 1 {
		t.Fatalf("expected cursor 1, got %d", m.settingsCursor)
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.settingsCursor != 0 {
		t.Fatalf("expected cursor 0, got %d", m.settingsCursor)
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyTab})
	if m.focus != focusSidebar {
		t.Fatalf("expected sidebar focus, got %d", m.focus)
	}
}

func TestPodcastSearchEnterPreviews(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = activate(m, viewPodcastSearch)
	m = updateModel(m, podcastSearchMsg{results: []podcast.SearchResult{{Title: "Show", URL: "http://feed"}}})

	// Enter triggers an async feed fetch; simulate the fetch completing.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModel(m, podcastFeedMsg{feed: &podcast.Feed{Title: "Show", URL: "http://feed"}, subscribe: false})

	if m.view != viewPodcastPreview {
		t.Fatalf("expected viewPodcastPreview, got %s", m.view)
	}
	if m.podcastPreviewFeed == nil {
		t.Fatal("expected feed preview to start")
	}
}

func TestPodcastPreviewViewTransitions(t *testing.T) {
	m, fp, _ := newTestModel(t)
	m = activate(m, viewPodcastSearch)
	m = updateModel(m, podcastSearchMsg{results: []podcast.SearchResult{{Title: "Show", URL: "http://feed"}}})

	// Enter on a search result opens the preview view.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModel(m, podcastFeedMsg{feed: &podcast.Feed{
		Title: "Show",
		URL:   "http://feed",
		Episodes: []podcast.Episode{
			{Title: "Episode 1", URL: "http://ep1"},
		},
	}, subscribe: false})

	if m.view != viewPodcastPreview {
		t.Fatalf("expected viewPodcastPreview, got %s", m.view)
	}

	// Enter in the preview plays the episode.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(fp.played) != 1 || fp.played[0].Name != "Episode 1" {
		t.Fatalf("expected episode to play, got %+v", fp.played)
	}

	// Esc returns to search view.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.view != viewPodcastSearch {
		t.Fatalf("expected viewPodcastSearch after Esc, got %s", m.view)
	}
	if m.podcastPreviewFeed != nil {
		t.Fatal("expected preview feed cleared")
	}
}

func TestPreviewEnterPlaysEvenWithStaleSelectedFeed(t *testing.T) {
	m, fp, _ := newTestModel(t)
	m = activate(m, viewPodcastSearch)

	// Pretend a subscription was opened earlier.
	m.podcasts = append(m.podcasts, podcast.Feed{Title: "Stale", URL: "http://stale"})
	m.podcastSelectedFeed = &m.podcasts[0]

	m = updateModel(m, podcastSearchMsg{results: []podcast.SearchResult{{Title: "Show", URL: "http://feed"}}})

	// Open the preview.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = updateModel(m, podcastFeedMsg{feed: &podcast.Feed{
		Title: "Show",
		URL:   "http://feed",
		Episodes: []podcast.Episode{
			{Title: "Episode 1", URL: "http://ep1"},
		},
	}, subscribe: false})

	// Enter in the preview must play from the preview, not the stale selected feed.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if len(fp.played) != 1 || fp.played[0].Name != "Episode 1" {
		t.Fatalf("expected episode to play from preview, got %+v", fp.played)
	}
}

func TestBuildPreviewItems(t *testing.T) {
	feed := &podcast.Feed{
		Title:       "My Show",
		URL:         "http://feed",
		Description: "A short description.",
		LastFetched: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Episodes: []podcast.Episode{
			{Title: "Old", URL: "http://old", PubDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Duration: 100},
			{Title: "New", URL: "http://new", PubDate: time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC), Duration: 200},
		},
	}

	m := model{}
	items := m.buildPreviewItems(feed, 80)

	if len(items) != 7 { // title header + blank + desc line + meta + separator + 2 episodes
		t.Fatalf("expected 7 preview items, got %d", len(items))
	}
	if items[0].Name != "Preview: My Show" || !items[0].IsHeader {
		t.Fatalf("unexpected title item: %+v", items[0])
	}
	if items[1].Name != "" || items[1].Kind != ui.KindPreviewInfo {
		t.Fatalf("expected blank line after title, got %+v", items[1])
	}
	if items[2].Name != "A short description." || items[2].Kind != ui.KindPreviewInfo {
		t.Fatalf("unexpected description item: %+v", items[2])
	}
	if items[3].Kind != ui.KindPreviewMeta {
		t.Fatalf("expected metadata item, got %+v", items[3])
	}
	if items[4].Kind != ui.KindPreviewMeta {
		t.Fatalf("expected separator item, got %+v", items[4])
	}
	if items[5].Name != "New" || items[5].URL != "http://new" {
		t.Fatalf("expected newest episode first, got %+v", items[5])
	}
	if items[6].Name != "Old" || items[6].URL != "http://old" {
		t.Fatalf("expected older episode second, got %+v", items[6])
	}
}

func TestBuildPreviewItemsCapsAtTen(t *testing.T) {
	feed := &podcast.Feed{Title: "Many", URL: "http://feed"}
	for i := 0; i < 25; i++ {
		feed.Episodes = append(feed.Episodes, podcast.Episode{
			Title:   fmt.Sprintf("Ep %d", i),
			URL:     fmt.Sprintf("http://ep%d", i),
			PubDate: time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC),
		})
	}
	m := model{}
	items := m.buildPreviewItems(feed, 80)

	epCount := 0
	for _, it := range items {
		if it.Kind == ui.KindPodcastEpisode {
			epCount++
		}
	}
	if epCount != 10 {
		t.Fatalf("expected 10 episodes, got %d", epCount)
	}
}

func TestActivateViewSetsFocus(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = activate(m, viewRadioFavourites)
	if m.view != viewRadioFavourites {
		t.Fatalf("expected favourites view, got %s", m.view)
	}
	if m.focus != focusList {
		t.Fatalf("expected list focus, got %d", m.focus)
	}
}

func TestSeekKeysSeekPodcast(t *testing.T) {
	m, fp, _ := newTestModel(t)
	m.status.Media = player.Media{Name: "Episode", URL: "http://ep", IsPodcast: true}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyShiftLeft})
	if len(fp.seeked) != 1 || fp.seeked[0] != -10 {
		t.Fatalf("expected seek -10, got %v", fp.seeked)
	}
	if m.toast != "Seek -10s" {
		t.Fatalf("expected toast, got %q", m.toast)
	}

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyShiftRight})
	if len(fp.seeked) != 2 || fp.seeked[1] != 10 {
		t.Fatalf("expected seek +10, got %v", fp.seeked)
	}
}

func TestSaveFavouriteShowsToast(t *testing.T) {
	m, _, _ := newTestModel(t)
	m.saving = true
	m.saveStation = store.Station{Name: "Soma", URL: "http://soma"}
	m.saveCat.SetValue("chill")

	m = updateModel(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.toast != "Saved to favourites" {
		t.Fatalf("expected toast, got %q", m.toast)
	}
	if m.saving {
		t.Fatal("expected dialog closed")
	}
}

func TestSpaceTogglesPauseOnlyWhenLoaded(t *testing.T) {
	m, fp, _ := newTestModel(t)

	// No media loaded: Space does nothing.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeySpace})
	if fp.toggleCnt != 0 {
		t.Fatalf("expected no toggle when idle, got %d", fp.toggleCnt)
	}

	// Podcast media loaded: Space toggles.
	m.status.Media = player.Media{Name: "Episode", URL: "http://ep", IsPodcast: true}
	m = updateModel(m, tea.KeyMsg{Type: tea.KeySpace})
	if fp.toggleCnt != 1 {
		t.Fatalf("expected one toggle, got %d", fp.toggleCnt)
	}
}

func TestRadioSpaceConnectsAndDisconnects(t *testing.T) {
	m, fp, _ := newTestModel(t)

	// Start playing a radio station.
	m.player.Play(player.Media{Name: "Station", URL: "http://station"})
	m.status.Media = player.Media{Name: "Station", URL: "http://station"}
	m.status.Playing = true

	// Space disconnects.
	m = updateModel(m, tea.KeyMsg{Type: tea.KeySpace})
	if fp.playing {
		t.Fatal("expected radio to disconnect")
	}

	// Space reconnects.
	m.status.Playing = false
	playedBefore := len(fp.played)
	m = updateModel(m, tea.KeyMsg{Type: tea.KeySpace})
	if len(fp.played) != playedBefore+1 {
		t.Fatalf("expected radio to reconnect, played count %d", len(fp.played))
	}
}
