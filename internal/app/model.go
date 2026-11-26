package app

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
	"github.com/mattn/go-runewidth"

	"github.com/daum3ns/zuelose/internal/player"
	"github.com/daum3ns/zuelose/internal/podcast"
	"github.com/daum3ns/zuelose/internal/radio"
	"github.com/daum3ns/zuelose/internal/store"
	"github.com/daum3ns/zuelose/internal/ui"
)

func init() {
	cfg := store.DefaultConfig()
	ui.SetTheme(ui.ThemeByName(cfg.Theme))
}

func themeForConfig(cfg store.Config) ui.Theme {
	t := ui.ThemeByName(cfg.Theme)
	if t.Mode != cfg.ThemeMode {
		return ui.DefaultThemeForMode(cfg.ThemeMode)
	}
	return t
}

type focusArea int

const (
	focusSidebar focusArea = iota
	focusSearch
	focusList
)

type pane int

const (
	paneSidebar pane = iota
	paneMain
)

type resumePrompt struct {
	feedURL    string
	episodeURL string
}

// Player controls audio playback.
type Player interface {
	Start() error
	Play(media player.Media)
	Stop()
	TogglePause()
	Seek(seconds float64)
	VolumeUp()
	VolumeDown()
	SetVolume(int)
}

// RadioClient searches for radio stations.
type RadioClient interface {
	Search(ctx context.Context, name string, limit int) ([]store.Station, error)
}

type searchMsg struct {
	stations []store.Station
	err      error
}

type podcastSearchMsg struct {
	results []podcast.SearchResult
	err     error
}

type podcastFeedMsg struct {
	feed      *podcast.Feed
	err       error
	subscribe bool
}

type playerReadyMsg struct {
	err error
}

type podcastRefreshTickMsg struct{}

type blinkMsg struct{}

type clearToastMsg struct{}

// model is the root Bubble Tea model for zuelose.
type model struct {
	width, height int
	config        store.Config
	favs          store.Favourites
	podcasts      []podcast.Feed
	client        RadioClient
	player        Player

	view          view
	focus         focusArea
	lastMainFocus focusArea
	sidebarItems  []sidebarItem
	sidebarCursor int

	// Radio browse.
	query     textinput.Model
	searching bool
	stations  []store.Station
	list      ui.List

	// Podcasts.
	podcastQuery         textinput.Model
	podcastAddInput      textinput.Model
	podcastSearchResults []podcast.SearchResult
	podcastSearching     bool
	podcastSelectedFeed  *podcast.Feed
	podcastPreviewFeed   *podcast.Feed
	podcastEpisodeList   ui.List
	subscriptionCursor   int
	addingPodcast        bool
	resumePrompt         *resumePrompt

	// Create category dialog.
	creatingCategory bool
	catInput         textinput.Model

	// Save/edit favourite dialog.
	saving            bool
	saveFocusCategory bool
	saveStation       store.Station
	saveOrigCategory  string
	saveCat           textinput.Model

	// Settings editor.
	settingsCursor int
	editingSetting bool
	settingInput   textinput.Model

	// Playback state.
	status         player.StatusMsg
	currentFeed    *podcast.Feed
	currentEpisode *podcast.Episode
	pausedBlink    bool
	errMsg         string
	help           bool

	// Feedback toast.
	toast   string
	toastAt time.Time
}

// NewModel builds the initial application model.
func NewModel(cfg store.Config, favs store.Favourites, mpv Player, client RadioClient) model {
	q := textinput.New()
	q.Placeholder = "Search stations..."
	q.CharLimit = 64

	pq := textinput.New()
	pq.Placeholder = "Search podcasts..."
	pq.CharLimit = 64

	pa := textinput.New()
	pa.Placeholder = "https://example.com/feed.xml"
	pa.CharLimit = 256

	sc := textinput.New()
	sc.Placeholder = "category"
	sc.CharLimit = 32

	ci := textinput.New()
	ci.Placeholder = "new category name"
	ci.CharLimit = 32

	si := textinput.New()
	si.CharLimit = 128

	items := newSidebar()
	cursor := findSidebarIndex(items, viewRadioBrowse)

	m := model{
		config:          cfg,
		favs:            favs,
		client:          client,
		player:          mpv,
		view:            viewRadioBrowse,
		focus:           focusList,
		lastMainFocus:   focusList,
		sidebarItems:    items,
		sidebarCursor:   cursor,
		query:           q,
		podcastQuery:    pq,
		podcastAddInput: pa,
		saveCat:         sc,
		catInput:        ci,
		settingInput:    si,
	}
	m.podcasts, _ = store.LoadPodcasts()
	m.list.Items = m.buildStationItems()
	m.blurAllInputs()
	ui.SetTheme(themeForConfig(cfg))
	return m
}

// Init runs startup commands.
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.startPlayerCmd(),
		searchCmd(m.client, ""),
	}
	if len(m.podcasts) > 0 {
		cmds = append(cmds, m.refreshPodcastsCmd())
	}
	cmds = append(cmds, m.podcastRefreshTickCmd())
	return tea.Batch(cmds...)
}

// Update handles incoming messages and keyboard input.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInputs()
		return m, nil

	case player.StatusMsg:
		wasPaused := m.status.Paused
		m.status = msg
		if msg.Error != "" {
			m.errMsg = msg.Error
		}
		if msg.Position > 0 {
			if _, ep := m.findCurrentEpisode(); ep != nil {
				ep.Position = msg.Position
			}
		}
		if msg.Paused && !wasPaused {
			return m, blinkCmd()
		}
		if !msg.Paused {
			m.pausedBlink = false
		}
		return m, nil

	case blinkMsg:
		if m.status.Paused {
			m.pausedBlink = !m.pausedBlink
			return m, blinkCmd()
		}
		m.pausedBlink = false
		return m, nil

	case clearToastMsg:
		m.toast = ""
		return m, nil

	case player.FinishedMsg:
		if _, ep := m.findCurrentEpisode(); ep != nil {
			ep.Completed = true
			ep.Position = 0
			ep.Played = true
			ep.IsNew = false
			_ = m.savePodcasts()
		}
		return m, nil

	case searchMsg:
		m.searching = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.stations = nil
		} else {
			m.stations = msg.stations
		}
		if m.view == viewRadioBrowse {
			m.list.Items = m.buildStationItems()
		}
		return m, nil

	case podcastSearchMsg:
		m.podcastSearching = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.podcastSearchResults = nil
		} else {
			m.podcastSearchResults = msg.results
		}
		if m.view == viewPodcastSearch {
			m.podcastPreviewFeed = nil
			m.podcastEpisodeList.Items = m.buildPodcastSearchItems()
		}
		return m, nil

	case podcastFeedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		feed := msg.feed
		feed.LastFetched = time.Now()
		if msg.subscribe {
			// Merge with existing subscription if present.
			existingIdx := -1
			for i, f := range m.podcasts {
				if f.URL == feed.URL {
					existingIdx = i
					break
				}
			}
			isNew := existingIdx < 0
			if isNew {
				m.podcasts = append(m.podcasts, *feed)
			} else {
				// Preserve episode state on refresh.
				feed.Episodes = mergeEpisodes(feed.Episodes, m.podcasts[existingIdx].Episodes)
				m.podcasts[existingIdx] = *feed
			}
			_ = m.savePodcasts()
			m.podcastSelectedFeed = feed
			if m.view == viewPodcastSubscriptions {
				m.podcastEpisodeList.Items = m.buildSubscriptionItems()
			}
			if m.view == viewPodcastNewEpisodes {
				m.podcastEpisodeList.Items = m.buildNewEpisodeItems()
			}
			if m.view == viewPodcastContinueListening {
				m.podcastEpisodeList.Items = m.buildContinueListeningItems()
			}
			if m.view == viewPodcastPreview && m.podcastPreviewFeed != nil && m.podcastPreviewFeed.URL == feed.URL {
				m.podcastEpisodeList.Items = m.buildPreviewItems(m.podcastPreviewFeed, m.mainContentWidth())
				m.podcastEpisodeList.SelectFirst()
			}
			if !isNew {
				// Background refresh of an existing feed: no toast, no view change.
				return m, nil
			}
			if m.view == viewPodcastSearch {
				m.view = viewPodcastNewEpisodes
				m.focus = focusList
				m.podcastEpisodeList.Items = m.buildNewEpisodeItems()
			}
			return m.showToast("Subscribed to " + feed.Title)
		} else {
			// Preview from search without subscribing.
			m.podcastPreviewFeed = feed
			m.view = viewPodcastPreview
			m.focus = focusList
			// The list is rendered inside the main pane; use its inner width.
			m.podcastEpisodeList.Items = m.buildPreviewItems(feed, m.mainContentWidth())
			m.podcastEpisodeList.SelectFirst()
		}
		return m, nil

	case playerReadyMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		}
		m.player.SetVolume(m.config.Volume)
		return m, m.podcastRefreshTickCmd()

	case podcastRefreshTickMsg:
		return m, tea.Batch(m.refreshPodcastsCmd(), m.podcastRefreshTickCmd())

	case tea.KeyMsg:
		if m.help {
			if msg.String() == "?" || msg.Type == tea.KeyEsc || msg.String() == "q" {
				m.help = false
			}
			return m, nil
		}
		if m.saving {
			return m.updateSaveDialog(msg)
		}
		if m.creatingCategory {
			return m.updateCreateCategoryDialog(msg)
		}
		if m.addingPodcast {
			return m.updateAddPodcastDialog(msg)
		}
		if m.editingSetting {
			return m.updateSettingsDialog(msg)
		}
		if m.resumePrompt != nil {
			return m.updateResumePrompt(msg)
		}
		return m.updateNormal(msg)
	}

	return m, nil
}

// View renders the full UI.
func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	headerH := 2
	hintH := 1
	footerH := 7
	bodyH := m.height - headerH - footerH - hintH
	if bodyH < 1 {
		bodyH = 1
	}
	sidebarW := 24
	mainW := m.width - sidebarW
	if mainW < 10 {
		mainW = 10
	}

	header := m.renderHeader(headerH)
	hintBar := m.renderHintBar(hintH)
	sidebar := m.renderSidebar(bodyH, sidebarW)
	main := m.renderMain(bodyH, mainW)
	footer := m.renderFooter(footerH)

	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
	screen := lipgloss.JoinVertical(lipgloss.Left, header, hintBar, body, footer)

	if m.help {
		overlay := ui.HelpText()
		screen = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
	}

	if m.saving {
		dialog := m.renderSaveDialog()
		screen = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.creatingCategory {
		dialog := m.renderCreateCategoryDialog()
		screen = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.addingPodcast {
		dialog := m.renderAddPodcastDialog()
		screen = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.editingSetting {
		dialog := m.renderSettingsDialog()
		screen = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.resumePrompt != nil {
		dialog := m.renderResumePrompt()
		screen = lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return screen
}

// --- Commands ---

func (m model) startPlayerCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.player.Start(); err != nil {
			return playerReadyMsg{err: err}
		}
		return playerReadyMsg{}
	}
}

func searchCmd(client RadioClient, query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stations, err := client.Search(ctx, query, 50)
		return searchMsg{stations: stations, err: err}
	}
}

func podcastSearchCmd(term string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		results, err := podcast.SearchITunes(ctx, term)
		return podcastSearchMsg{results: results, err: err}
	}
}

func podcastFetchCmd(feedURL string, subscribe bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		feed, err := podcast.ParseFeed(ctx, feedURL)
		return podcastFeedMsg{feed: feed, err: err, subscribe: subscribe}
	}
}

func (m model) refreshPodcastsCmd() tea.Cmd {
	if len(m.podcasts) == 0 {
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(m.podcasts))
	for i := range m.podcasts {
		cmds = append(cmds, podcastFetchCmd(m.podcasts[i].URL, true))
	}
	return tea.Batch(cmds...)
}

func (m model) podcastRefreshTickCmd() tea.Cmd {
	interval := time.Duration(m.config.PodcastRefreshInterval) * time.Minute
	if interval <= 0 {
		interval = 60 * time.Minute
	}
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return podcastRefreshTickMsg{}
	})
}

func blinkCmd() tea.Cmd {
	return tea.Tick(600*time.Millisecond, func(time.Time) tea.Msg {
		return blinkMsg{}
	})
}

// --- Update helpers ---

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	m.errMsg = ""

	switch msg.Type {
	case tea.KeyTab:
		return m.cyclePane(1)
	case tea.KeyShiftTab:
		return m.cyclePane(-1)
	}

	// Global controls (unless typing in a text field).
	if !m.anyInputFocused() {
		switch k {
		case "right":
			return m.cyclePane(1)
		case "left":
			return m.cyclePane(-1)
		case "?":
			m.help = true
			return m, nil
		case "q", "ctrl+c":
			_ = m.savePodcasts()
			return m, tea.Quit
		case "+", "=":
			m.player.VolumeUp()
			return m, nil
		case "-":
			m.player.VolumeDown()
			return m, nil
		case " ":
			if m.status.Media.URL != "" || m.status.Playing || m.status.Paused {
				m.player.TogglePause()
			}
			return m, nil
		case "shift+left":
			m.player.Seek(-10)
			return m.showToast("Seek -10s")
		case "shift+right":
			m.player.Seek(10)
			return m.showToast("Seek +10s")
		case "shift+up":
			m.player.Seek(-60)
			return m.showToast("Seek -60s")
		case "shift+down":
			m.player.Seek(60)
			return m.showToast("Seek +60s")
		case "/":
			if m.view == viewRadioBrowse || m.view == viewPodcastSearch {
				m.focus = focusSearch
				m.focusCurrentSearch()
				return m, nil
			}
			if m.view == viewPodcastPreview {
				m.view = viewPodcastSearch
				m.podcastPreviewFeed = nil
				m.podcastEpisodeList.Items = m.buildPodcastSearchItems()
				m.focus = focusSearch
				m.focusCurrentSearch()
				return m, nil
			}
		case "r":
			if m.view == viewRadioBrowse {
				m.searching = true
				return m, searchCmd(m.client, m.query.Value())
			}
			if m.view == viewPodcastSearch {
				m.podcastSearching = true
				return m, podcastSearchCmd(m.podcastQuery.Value())
			}
		}
	}

	switch m.focus {
	case focusSidebar:
		return m.updateSidebar(msg)
	case focusSearch:
		return m.updateSearch(msg)
	case focusList:
		return m.updateList(msg)
	}

	return m, nil
}

func (m model) activePane() pane {
	switch m.focus {
	case focusSidebar:
		return paneSidebar
	default:
		return paneMain
	}
}

func (m model) paneBorderColor(p pane) lipgloss.Color {
	if m.activePane() == p {
		return ui.ColorPrimary()
	}
	return ui.ColorMuted()
}

// withDotBorder replaces a rendered pane's top border with a dot marker
// indicating whether the pane is focused.
func (m model) withDotBorder(view string, p pane) string {
	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		return view
	}
	marker := "○"
	if m.activePane() == p {
		marker = "●"
	}
	fill := lipgloss.Width(lines[0]) - 6 // "╭─ ● " + fill + "╮"
	if fill < 0 {
		fill = 0
	}
	top := lipgloss.NewStyle().
		Background(ui.ColorBg()).
		Foreground(m.paneBorderColor(p)).
		Render("╭─ " + marker + " " + strings.Repeat("─", fill) + "╮")
	lines[0] = top
	return strings.Join(lines, "\n")
}

func (m model) viewSupportsSearch() bool {
	return m.view == viewRadioBrowse || m.view == viewPodcastSearch
}

func (m model) cyclePane(direction int) (tea.Model, tea.Cmd) {
	if m.focus == focusSearch || m.focus == focusList {
		m.lastMainFocus = m.focus
	}

	panes := []pane{paneSidebar, paneMain}

	idx := 0
	current := m.activePane()
	for i, p := range panes {
		if p == current {
			idx = i
			break
		}
	}
	idx = (idx + direction + len(panes)) % len(panes)
	return m.focusPane(panes[idx])
}

func (m model) focusPane(p pane) (tea.Model, tea.Cmd) {
	switch p {
	case paneSidebar:
		m.blurAllInputs()
		m.focus = focusSidebar
		return m, nil
	case paneMain:
		m.blurAllInputs()
		if m.lastMainFocus == focusSearch && m.viewSupportsSearch() {
			m.focus = focusSearch
			m.focusCurrentSearch()
		} else {
			m.focus = focusList
		}
		return m, nil
	}
	return m, nil
}

func (m model) anyInputFocused() bool {
	return m.query.Focused() ||
		m.podcastQuery.Focused() ||
		m.podcastAddInput.Focused() ||
		m.saveCat.Focused() ||
		m.catInput.Focused() ||
		m.settingInput.Focused()
}

func (m model) updateSidebar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		prev := m.sidebarCursor
		for m.sidebarCursor > 0 {
			m.sidebarCursor--
			if !m.sidebarItems[m.sidebarCursor].isHeader {
				break
			}
		}
		if m.sidebarItems[m.sidebarCursor].isHeader {
			m.sidebarCursor = prev
		}
	case tea.KeyDown:
		prev := m.sidebarCursor
		for m.sidebarCursor < len(m.sidebarItems)-1 {
			m.sidebarCursor++
			if !m.sidebarItems[m.sidebarCursor].isHeader {
				break
			}
		}
		if m.sidebarItems[m.sidebarCursor].isHeader {
			m.sidebarCursor = prev
		}
	case tea.KeyEnter:
		item := m.sidebarItems[m.sidebarCursor]
		if !item.isHeader {
			return m.activateView(item.v)
		}
	case tea.KeyEsc:
		m.focus = focusList
		m.blurAllInputs()
		return m, nil
	}
	return m, nil
}

func (m model) activateView(v view) (tea.Model, tea.Cmd) {
	m.view = v
	m.blurAllInputs()
	m.editingSetting = false

	var cmd tea.Cmd
	switch v {
	case viewRadioBrowse:
		m.focus = focusList
		m.blurAllInputs()
		m.podcastPreviewFeed = nil
		m.list.Items = m.buildStationItems()
	case viewPodcastSearch:
		m.focus = focusSearch
		m.focusCurrentSearch()
		m.podcastPreviewFeed = nil
		m.podcastEpisodeList.Items = m.buildPodcastSearchItems()
	case viewPodcastPreview:
		m.focus = focusList
		m.blurAllInputs()
		if m.podcastPreviewFeed != nil {
			m.podcastEpisodeList.Items = m.buildPreviewItems(m.podcastPreviewFeed, m.mainContentWidth())
			m.podcastEpisodeList.SelectFirst()
		}
	case viewRadioFavourites:
		m.focus = focusList
		m.blurAllInputs()
		m.podcastPreviewFeed = nil
		m.list.Items = m.buildFavItems()
		m.list.SelectFirst()
	case viewPodcastSubscriptions:
		m.focus = focusList
		m.blurAllInputs()
		m.podcastPreviewFeed = nil
		m.podcastEpisodeList.Items = m.buildSubscriptionItems()
		cmd = m.refreshPodcastsCmd()
	case viewPodcastNewEpisodes:
		m.focus = focusList
		m.blurAllInputs()
		m.podcastPreviewFeed = nil
		m.podcastEpisodeList.Items = m.buildNewEpisodeItems()
	case viewPodcastContinueListening:
		m.focus = focusList
		m.blurAllInputs()
		m.podcastPreviewFeed = nil
		m.podcastEpisodeList.Items = m.buildContinueListeningItems()
	}
	if m.focus == focusSearch || m.focus == focusList {
		m.lastMainFocus = m.focus
	}
	return m, cmd
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// The search input is only used in the actual search views.
	if m.view != viewRadioBrowse && m.view != viewPodcastSearch {
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEnter:
		m.focus = focusList
		m.blurAllInputs()
		if m.view == viewRadioBrowse {
			m.searching = true
			return m, searchCmd(m.client, m.query.Value())
		}
		if m.view == viewPodcastSearch {
			m.podcastPreviewFeed = nil
			m.podcastSearching = true
			return m, podcastSearchCmd(m.podcastQuery.Value())
		}
		return m, nil
	case tea.KeyDown:
		m.focus = focusList
		m.blurAllInputs()
		m.currentList().Down()
		return m, nil
	case tea.KeyEsc:
		m.focus = focusList
		m.blurAllInputs()
		return m, nil
	}

	var cmd tea.Cmd
	if m.view == viewRadioBrowse {
		m.query, cmd = m.query.Update(msg)
	} else if m.view == viewPodcastSearch {
		m.podcastQuery, cmd = m.podcastQuery.Update(msg)
	}
	return m, cmd
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()

	// Settings uses its own cursor, not a list.
	if m.view == viewSettings {
		switch k {
		case "up":
			if m.settingsCursor > 0 {
				m.settingsCursor--
			}
			return m, nil
		case "down":
			if m.settingsCursor < m.settingsCount()-1 {
				m.settingsCursor++
			}
			return m, nil
		case "enter", "e":
			switch m.settingsCursor {
			case 0, 1, 2, 3:
				return m.openSettingsEditor()
			case 4:
				return m.cyclePodcastBrowser()
			case 5:
				return m.cycleThemeMode()
			case 6:
				return m.cycleTheme()
			}
			return m, nil
		case "esc":
			m.focus = focusSidebar
			m.blurAllInputs()
			return m, nil
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyUp:
		m.currentList().Up()
		return m, nil
	case tea.KeyDown:
		m.currentList().Down()
		return m, nil
	case tea.KeyEnter:
		return m.activateSelected()
	case tea.KeyEsc:
		switch m.view {
		case viewPodcastPreview:
			m.podcastPreviewFeed = nil
			m.view = viewPodcastSearch
			m.podcastEpisodeList.Items = m.buildPodcastSearchItems()
			m.podcastEpisodeList.Cursor = 0
			return m, nil
		case viewPodcastEpisodes:
			m.view = viewPodcastSubscriptions
			m.podcastEpisodeList.Items = m.buildSubscriptionItems()
			if m.subscriptionCursor >= 0 && m.subscriptionCursor < len(m.podcastEpisodeList.Items) {
				m.podcastEpisodeList.Cursor = m.subscriptionCursor
			} else {
				m.podcastEpisodeList.Cursor = 0
			}
			return m, nil
		}
		m.focus = focusSidebar
		m.blurAllInputs()
		return m, nil
	}

	switch m.view {
	case viewRadioBrowse:
		if k == "f" {
			return m.openSaveDialog(false)
		}
	case viewRadioFavourites:
		switch k {
		case "d":
			return m.deleteSelectedFavourite()
		case "f":
			return m.openSaveDialog(true)
		case "C":
			m.creatingCategory = true
			m.catInput.Focus()
			m.catInput.SetValue("")
			return m, textinput.Blink
		}
	case viewPodcastSearch:
		switch k {
		case "a":
			return m.subscribeToSelectedSearchResult()
		}
	case viewPodcastPreview:
		switch k {
		case "a":
			return m.subscribePreviewedFeed()
		}
	case viewPodcastSubscriptions:
		switch k {
		case "a":
			m.addingPodcast = true
			m.podcastAddInput.Focus()
			m.podcastAddInput.SetValue("")
			return m, textinput.Blink
		case "d":
			return m.deleteSelectedSubscription()
		case "r":
			return m, m.refreshPodcastsCmd()
		}
	case viewPodcastNewEpisodes:
		if k == "r" {
			return m, m.refreshPodcastsCmd()
		}
	case viewPodcastContinueListening:
		if k == "r" {
			return m, m.refreshPodcastsCmd()
		}
	}

	return m, nil
}

func (m *model) currentList() *ui.List {
	switch m.view {
	case viewRadioBrowse, viewRadioFavourites:
		return &m.list
	case viewPodcastSearch, viewPodcastPreview, viewPodcastSubscriptions, viewPodcastEpisodes, viewPodcastNewEpisodes, viewPodcastContinueListening:
		return &m.podcastEpisodeList
	}
	return &m.list
}

func (m model) activateSelected() (tea.Model, tea.Cmd) {
	switch m.view {
	case viewRadioBrowse:
		return m.playSelectedStation()
	case viewRadioFavourites:
		return m.playSelectedFavourite()
	case viewPodcastSearch:
		return m.openSelectedSearchResult()
	case viewPodcastPreview:
		return m.playSelectedEpisode(false, false)
	case viewPodcastSubscriptions:
		return m.openSelectedSubscription()
	case viewPodcastEpisodes:
		return m.playSelectedEpisode(false, false)
	case viewPodcastNewEpisodes:
		return m.playSelectedEpisode(false, false)
	case viewPodcastContinueListening:
		return m.resumeSelectedEpisode()
	}
	return m, nil
}

func (m model) playSelectedStation() (tea.Model, tea.Cmd) {
	sel := m.list.Selected()
	if sel == nil || sel.IsHeader {
		return m, nil
	}
	idx := m.list.Cursor
	if idx < 0 || idx >= len(m.stations) {
		return m, nil
	}
	st := m.stations[idx]
	m.savePodcastPosition()
	m.currentEpisode = nil
	m.currentFeed = nil
	m.player.Play(player.Media{
		Name:    st.Name,
		URL:     st.PlayURL(),
		Codec:   strings.ToUpper(st.Codec),
		Bitrate: st.Bitrate,
		Country: st.CountryCode,
		Tags:    st.Tags,
	})
	return m, nil
}

func (m model) playSelectedFavourite() (tea.Model, tea.Cmd) {
	sel := m.list.Selected()
	if sel == nil || sel.IsHeader {
		return m, nil
	}
	cat := m.selectedFavCategory()
	for _, c := range m.favs.Categories {
		if c.Name != cat {
			continue
		}
		for _, fav := range c.Stations {
			if fav.Name == sel.Name {
				m.savePodcastPosition()
				m.currentEpisode = nil
				m.currentFeed = nil
				m.player.Play(player.Media{
					Name:    fav.Name,
					URL:     fav.PlayURL(),
					Codec:   strings.ToUpper(fav.Codec),
					Bitrate: fav.Bitrate,
					Country: fav.CountryCode,
					Tags:    fav.Tags,
				})
				return m, nil
			}
		}
	}
	return m, nil
}

func (m model) openSelectedSearchResult() (tea.Model, tea.Cmd) {
	sel := m.podcastEpisodeList.Selected()
	if sel == nil {
		return m, nil
	}
	idx := m.podcastEpisodeList.Cursor
	if idx < 0 || idx >= len(m.podcastSearchResults) {
		return m, nil
	}
	res := m.podcastSearchResults[idx]
	return m, podcastFetchCmd(res.URL, false)
}

func (m model) subscribeToSelectedSearchResult() (tea.Model, tea.Cmd) {
	sel := m.podcastEpisodeList.Selected()
	if sel == nil {
		return m, nil
	}
	idx := m.podcastEpisodeList.Cursor
	if idx < 0 || idx >= len(m.podcastSearchResults) {
		return m, nil
	}
	res := m.podcastSearchResults[idx]
	return m, podcastFetchCmd(res.URL, true)
}

func (m model) subscribePreviewedFeed() (tea.Model, tea.Cmd) {
	if m.podcastPreviewFeed == nil {
		return m, nil
	}
	return m, podcastFetchCmd(m.podcastPreviewFeed.URL, true)
}

func (m model) openSelectedSubscription() (tea.Model, tea.Cmd) {
	sel := m.podcastEpisodeList.Selected()
	if sel == nil {
		return m, nil
	}
	idx := m.podcastEpisodeList.Cursor
	if idx < 0 || idx >= len(m.podcasts) {
		return m, nil
	}
	feed := &m.podcasts[idx]
	m.podcastSelectedFeed = feed
	m.subscriptionCursor = idx
	m.view = viewPodcastEpisodes
	m.podcastEpisodeList.Items = m.buildPodcastEpisodeItems(feed)
	return m, nil
}

func (m model) deleteSelectedSubscription() (tea.Model, tea.Cmd) {
	sel := m.podcastEpisodeList.Selected()
	if sel == nil {
		return m, nil
	}
	idx := m.podcastEpisodeList.Cursor
	if idx < 0 || idx >= len(m.podcasts) {
		return m, nil
	}
	m.podcasts = append(m.podcasts[:idx], m.podcasts[idx+1:]...)
	_ = m.savePodcasts()
	m.podcastEpisodeList.Items = m.buildSubscriptionItems()
	return m.showToast("Unsubscribed")
}

func (m model) playSelectedEpisode(forceRestart, forceResume bool) (tea.Model, tea.Cmd) {
	sel := m.podcastEpisodeList.Selected()
	if sel == nil || sel.Kind != ui.KindPodcastEpisode {
		return m, nil
	}

	var feed *podcast.Feed
	var ep *podcast.Episode

	// Locate in subscriptions first.
	if sel.FeedURL != "" && sel.URL != "" {
		feed, ep = store.FindEpisode(m.podcasts, sel.FeedURL, sel.URL)
	}

	// Fallback to the appropriate feed for episodes not found in subscriptions.
	// In the preview view, the active preview feed must win over any stale
	// selected subscription feed, otherwise Enter in a preview can silently fail.
	if ep == nil {
		if m.view == viewPodcastPreview && m.podcastPreviewFeed != nil {
			feed = m.podcastPreviewFeed
		} else if m.podcastSelectedFeed != nil {
			feed = m.podcastSelectedFeed
		} else if m.podcastPreviewFeed != nil {
			feed = m.podcastPreviewFeed
		}
		if feed != nil {
			for i := range feed.Episodes {
				if feed.Episodes[i].URL == sel.URL {
					ep = &feed.Episodes[i]
					break
				}
			}
		}
	}

	if feed == nil || ep == nil {
		return m, nil
	}

	if !forceRestart && !forceResume && ep.Position > 0 && !ep.Completed {
		m.resumePrompt = &resumePrompt{feedURL: feed.URL, episodeURL: ep.URL}
		return m, nil
	}

	resumeFrom := 0
	if forceRestart {
		resumeFrom = 0
		ep.Position = 0
	} else if (forceResume || ep.Position > 0) && !ep.Completed {
		resumeFrom = ep.Position
	}

	m.savePodcastPosition()
	m.currentFeed = feed
	m.currentEpisode = ep
	ep.Played = true
	ep.Completed = false
	ep.IsNew = false
	_ = m.savePodcasts()

	m.player.Play(player.Media{
		Name:       ep.Title,
		URL:        ep.URL,
		Codec:      "PODCAST",
		ResumeFrom: resumeFrom,
		IsPodcast:  true,
	})
	return m, nil
}

func (m model) resumeSelectedEpisode() (tea.Model, tea.Cmd) {
	return m.playSelectedEpisode(false, true)
}

func (m model) updateResumePrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		m.resumePrompt = nil
		return m.playSelectedEpisode(false, true)
	case "enter", "s":
		m.resumePrompt = nil
		return m.playSelectedEpisode(true, false)
	case "esc":
		m.resumePrompt = nil
	}
	return m, nil
}

func (m *model) savePodcastPosition() {
	if m.currentEpisode != nil {
		_ = m.savePodcasts()
	}
}

func (m *model) findCurrentEpisode() (*podcast.Feed, *podcast.Episode) {
	if m.currentFeed == nil || m.currentEpisode == nil {
		return nil, nil
	}
	return store.FindEpisode(m.podcasts, m.currentFeed.URL, m.currentEpisode.URL)
}

func (m model) savePodcasts() error {
	return store.SavePodcasts(m.podcasts)
}

// --- Favourite dialogs ---

func (m model) openSaveDialog(edit bool) (tea.Model, tea.Cmd) {
	sel := m.list.Selected()
	if sel == nil || sel.IsHeader {
		return m, nil
	}

	var st store.Station
	if m.view == viewRadioBrowse {
		idx := m.list.Cursor
		if idx < 0 || idx >= len(m.stations) {
			return m, nil
		}
		st = m.stations[idx]
	} else if m.view == viewRadioFavourites && edit {
		cat := m.selectedFavCategory()
		for _, c := range m.favs.Categories {
			if c.Name != cat {
				continue
			}
			for _, fav := range c.Stations {
				if fav.Name == sel.Name {
					st = fav.Station
					m.saveOrigCategory = cat
					m.saveCat.SetValue(cat)
					m.saving = true
					m.saveFocusCategory = true
					m.saveStation = st
					m.saveCat.Focus()
					return m, textinput.Blink
				}
			}
		}
		return m, nil
	}

	m.saveStation = st
	m.saveOrigCategory = ""
	m.saveCat.SetValue("")
	m.saveFocusCategory = true
	m.saving = true
	m.saveCat.Focus()
	return m, textinput.Blink
}

func (m model) updateSaveDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.saving = false
		m.blurAllInputs()
		return m, nil
	case tea.KeyTab:
		m.autocompleteCategory()
		return m, nil
	case tea.KeyEnter:
		return m.submitSaveDialog()
	}

	var cmd tea.Cmd
	m.saveCat, cmd = m.saveCat.Update(msg)
	return m, cmd
}

func (m *model) autocompleteCategory() {
	cats := m.favs.CategoryNames()
	if len(cats) == 0 {
		return
	}
	current := strings.TrimSpace(m.saveCat.Value())
	idx := -1
	for i, c := range cats {
		if strings.EqualFold(c, current) {
			idx = i
			break
		}
	}
	prefix := strings.ToLower(current)
	start := 0
	if idx >= 0 {
		start = idx + 1
	}
	for i := 0; i < len(cats); i++ {
		ci := (start + i) % len(cats)
		if prefix == "" || strings.HasPrefix(strings.ToLower(cats[ci]), prefix) {
			m.saveCat.SetValue(cats[ci])
			return
		}
	}
	if idx >= 0 {
		m.saveCat.SetValue(cats[(idx+1)%len(cats)])
	} else {
		m.saveCat.SetValue(cats[0])
	}
}

func (m model) submitSaveDialog() (tea.Model, tea.Cmd) {
	cat := strings.TrimSpace(m.saveCat.Value())
	if cat == "" {
		m.errMsg = "category is required"
		return m, nil
	}

	if m.saveOrigCategory != "" {
		m.favs.Remove(m.saveOrigCategory, m.saveStation.Name)
	}

	if err := m.favs.Add(m.saveStation, cat); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	if err := store.SaveFavourites(m.favs); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}

	m.saving = false
	m.blurAllInputs()
	if m.view == viewRadioFavourites {
		m.list.Items = m.buildFavItems()
	}
	return m.showToast("Saved to favourites")
}

func (m model) deleteSelectedFavourite() (tea.Model, tea.Cmd) {
	sel := m.list.Selected()
	if sel == nil {
		return m, nil
	}
	if sel.IsHeader {
		for i, cat := range m.favs.Categories {
			if cat.Name != sel.Name {
				continue
			}
			if len(cat.Stations) > 0 {
				m.errMsg = "category is not empty"
				return m, nil
			}
			m.favs.Categories = append(m.favs.Categories[:i], m.favs.Categories[i+1:]...)
			if err := store.SaveFavourites(m.favs); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.list.Items = m.buildFavItems()
			return m.showToast("Category deleted")
		}
		return m, nil
	}
	cat := m.selectedFavCategory()
	if cat == "" {
		return m, nil
	}
	m.favs.Remove(cat, sel.Name)
	for i, c := range m.favs.Categories {
		if c.Name == cat && len(c.Stations) == 0 {
			m.favs.Categories = append(m.favs.Categories[:i], m.favs.Categories[i+1:]...)
			break
		}
	}
	if err := store.SaveFavourites(m.favs); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}
	m.list.Items = m.buildFavItems()
	return m.showToast("Removed from favourites")
}

func (m model) updateCreateCategoryDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.creatingCategory = false
		m.catInput.Blur()
		return m, nil
	case tea.KeyEnter:
		name := strings.TrimSpace(m.catInput.Value())
		if name == "" {
			m.errMsg = "category name cannot be empty"
			return m, nil
		}
		if err := m.favs.CreateCategory(name); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		if err := store.SaveFavourites(m.favs); err != nil {
			m.errMsg = err.Error()
			return m, nil
		}
		m.creatingCategory = false
		m.catInput.Blur()
		m.list.Items = m.buildFavItems()
		return m.showToast("Category created")
	}

	var cmd tea.Cmd
	m.catInput, cmd = m.catInput.Update(msg)
	return m, cmd
}

// --- Podcast add dialog ---

func (m model) updateAddPodcastDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.addingPodcast = false
		m.podcastAddInput.Blur()
		return m, nil
	case tea.KeyEnter:
		feedURL := strings.TrimSpace(m.podcastAddInput.Value())
		if feedURL == "" {
			m.errMsg = "feed URL cannot be empty"
			return m, nil
		}
		m.addingPodcast = false
		m.podcastAddInput.Blur()
		return m, podcastFetchCmd(feedURL, true)
	}

	var cmd tea.Cmd
	m.podcastAddInput, cmd = m.podcastAddInput.Update(msg)
	return m, cmd
}

// --- View helpers ---

// mainContentWidth returns the usable width inside the main pane's border.
func (m model) mainContentWidth() int {
	w := m.width - 24 - 2
	if w < 10 {
		w = 10
	}
	return w
}

func (m *model) resizeInputs() {
	if m.width <= 30 {
		return
	}
	mainContentW := m.mainContentWidth()

	queryW := mainContentW - 10
	if queryW < 1 {
		queryW = 1
	}
	m.query.Width = queryW
	m.podcastQuery.Width = queryW

	// Save dialog: box width m.width-10, padding 2 on each side.
	m.saveCat.Width = m.width - 14

	// Small dialogs: box width m.width-20, padding 2 on each side.
	m.podcastAddInput.Width = m.width - 24
	m.catInput.Width = m.width - 24
}

func (m *model) blurAllInputs() {
	m.query.Blur()
	m.podcastQuery.Blur()
	m.podcastAddInput.Blur()
	m.saveCat.Blur()
	m.catInput.Blur()
	m.settingInput.Blur()
}

func (m *model) focusCurrentSearch() {
	m.blurAllInputs()
	if m.view == viewRadioBrowse {
		m.query.Focus()
	} else if m.view == viewPodcastSearch || m.view == viewPodcastPreview {
		m.podcastQuery.Focus()
	}
}

func (m model) showToast(msg string) (tea.Model, tea.Cmd) {
	m.toast = msg
	m.toastAt = time.Now()
	return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearToastMsg{} })
}

func (m model) renderHeader(height int) string {
	title := "♪ zuelose"
	innerW := m.width - 2
	if innerW < 1 {
		innerW = 1
	}
	padding := (innerW - runewidth.StringWidth(title)) / 2
	if padding < 0 {
		padding = 0
	}
	titleLine := runewidth.Truncate(strings.Repeat(" ", padding)+title, innerW, "")
	sepLine := strings.Repeat("─", innerW)

	content := titleLine + "\n" + sepLine
	return ui.HeaderStyle.Height(height).MaxHeight(height).Width(m.width).Render(content)
}

func (m model) renderFooter(height int) string {
	disconnected := m.status.Media.URL != "" && !m.status.Media.IsPodcast && !m.status.Playing
	showAsPaused := m.status.Paused || disconnected

	name := "Stopped"
	if m.status.Media.Name != "" {
		name = m.status.Media.Name
	}
	pausedStyle := lipgloss.NewStyle().Foreground(ui.ColorWarn())
	if m.status.Paused {
		name = pausedStyle.Blink(true).Render(name + " (paused)")
	} else if disconnected {
		name = pausedStyle.Blink(true).Render(name + " (disconnected)")
	}

	codec := m.status.Codec
	if codec == "" {
		codec = "?"
	}
	bitrate := "?"
	if m.status.Bitrate > 0 {
		bitrate = fmt.Sprintf("%dk", m.status.Bitrate)
	}

	var textLines []string
	playbackHints := ""
	if m.status.Media.URL != "" {
		if m.status.Media.IsPodcast {
			playbackHints = "Space pause · Shift+←→ seek"
		} else if m.status.Playing {
			playbackHints = "Space disconnect"
		} else {
			playbackHints = "Space connect"
		}
	}

	if m.status.Media.IsPodcast {
		feedTitle := "Unknown podcast"
		if m.currentFeed != nil && m.currentFeed.Title != "" {
			feedTitle = m.currentFeed.Title
		}
		duration := 0
		if m.currentEpisode != nil {
			duration = m.currentEpisode.Duration
		}
		progress := m.renderProgressBar(m.status.Position, duration, 30)
		info := fmt.Sprintf("Podcast · %s / %s · Vol: %d%%",
			formatDuration(m.status.Position),
			formatDuration(duration),
			m.status.Volume)
		textLines = []string{
			name,
			feedTitle,
			progress,
			info,
		}
		if playbackHints != "" {
			textLines = append(textLines, ui.MutedStyle.Render(playbackHints))
		}
	} else {
		track := m.status.Title
		if track == "" {
			track = "Unknown track"
		}
		info := fmt.Sprintf("%s · %s · Vol: %d%%", codec, bitrate, m.status.Volume)
		textLines = []string{
			name,
			fmt.Sprintf("Track: %s", track),
			info,
		}
		if playbackHints != "" {
			textLines = append(textLines, ui.MutedStyle.Render(playbackHints))
		}
	}

	if showAsPaused {
		for i := 1; i < len(textLines); i++ {
			textLines[i] = pausedStyle.Render(textLines[i])
		}
	}

	if m.errMsg != "" {
		lastIdx := len(textLines) - 1
		textLines[lastIdx] = ui.ErrorStyle.Render(m.errMsg)
	}

	innerW := m.width - 2
	innerH := height - 2
	if innerW < 30 {
		innerW = 30
	}
	if innerH < 3 {
		innerH = 3
	}

	logoW := 24
	spacerW := 2
	textW := innerW - logoW - spacerW
	if textW < 10 {
		textW = 10
	}

	for i, line := range textLines {
		textLines[i] = runewidth.Truncate(line, textW, "")
	}

	initialsName := m.status.Media.Name
	if m.status.Media.IsPodcast && m.currentFeed != nil && m.currentFeed.Title != "" {
		initialsName = m.currentFeed.Title
	}
	initials := renderInitials(logoW, initialsName)
	initialsBox := lipgloss.NewStyle().
		Width(logoW).
		Height(innerH).
		MaxHeight(innerH).
		Background(ui.ColorBg()).
		Render(initials)
	spacer := lipgloss.NewStyle().Width(spacerW).Height(innerH).Background(ui.ColorBg()).Render("")

	textContent := strings.Join(textLines, "\n")
	textBlock := lipgloss.PlaceVertical(innerH, lipgloss.Center, textContent)
	textBox := lipgloss.NewStyle().Width(textW).Height(innerH).MaxHeight(innerH).Background(ui.ColorBg()).Render(textBlock)

	contentBlock := lipgloss.JoinHorizontal(lipgloss.Top, initialsBox, spacer, textBox)
	return m.renderFooterWithLabel(height, innerW, contentBlock)
}

func (m model) renderFooterWithLabel(height, innerW int, content string) string {
	label := "──── Now Playing "
	labelW := runewidth.StringWidth(label)
	fill := innerW - labelW
	if fill < 0 {
		fill = 0
	}

	borderStyle := lipgloss.NewStyle().Background(ui.ColorBg()).Foreground(ui.ColorMuted())
	labelStyle := lipgloss.NewStyle().Background(ui.ColorBg()).Foreground(ui.ColorPrimary())
	contentBg := lipgloss.NewStyle().Background(ui.ColorBg())

	top := borderStyle.Render("╭") + labelStyle.Render(label) + borderStyle.Render(strings.Repeat("─", fill)+"╮")
	bottom := borderStyle.Render("╰" + strings.Repeat("─", innerW) + "╯")

	contentLines := strings.Split(content, "\n")
	var lines []string
	lines = append(lines, top)
	for _, line := range contentLines {
		left := borderStyle.Render("│")
		right := borderStyle.Render("│")
		lines = append(lines, left+contentBg.Render(line)+right)
	}
	lines = append(lines, bottom)

	return lipgloss.NewStyle().
		Height(height).
		MaxHeight(height).
		Width(m.width).
		Background(ui.ColorBg()).
		Render(strings.Join(lines, "\n"))
}

func (m model) renderSidebar(height, width int) string {
	innerW := width - 2
	innerH := height - 2
	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	var lines []string
	for i, it := range m.sidebarItems {
		style := lipgloss.NewStyle().
			PaddingLeft(it.depth * 2).
			Width(innerW).
			MaxWidth(innerW)

		if it.isHeader {
			style = style.
				Background(ui.ColorSurface()).
				Foreground(ui.ColorSuccess()).
				Bold(true)
		}

		selected := i == m.sidebarCursor
		if !it.isHeader && selected && m.focus == focusSidebar {
			style = style.Background(ui.ColorPrimary()).Foreground(ui.ColorBg()).Bold(true)
		} else if !it.isHeader && selected {
			style = style.Background(ui.ColorSecondary()).Foreground(ui.ColorBg()).Bold(true)
		}

		lines = append(lines, style.Render(it.label))
	}
	content := strings.Join(lines, "\n")
	view := ui.SidebarStyle.
		BorderForeground(m.paneBorderColor(paneSidebar)).
		Height(height).
		MaxHeight(height).
		Width(width - 2).
		Render(content)
	return m.withDotBorder(view, paneSidebar)
}

func (m model) renderMain(height, width int) string {
	var content string
	switch m.view {
	case viewRadioBrowse:
		content = m.renderBrowse(height, width)
	case viewRadioFavourites:
		content = m.renderFavourites(height, width)
	case viewPodcastSearch:
		content = m.renderPodcastSearch(height, width)
	case viewPodcastPreview:
		content = m.renderPodcastPreview(height, width)
	case viewPodcastSubscriptions:
		content = m.renderPodcastSubscriptions(height, width)
	case viewPodcastEpisodes:
		content = m.renderPodcastEpisodes(height, width)
	case viewPodcastNewEpisodes:
		content = m.renderPodcastNewEpisodes(height, width)
	case viewPodcastContinueListening:
		content = m.renderPodcastContinueListening(height, width)
	case viewSettings:
		content = m.renderSettings(height, width)
	default:
		return ""
	}
	return m.withDotBorder(content, paneMain)
}

func (m model) mainPanelStyle() lipgloss.Style {
	return ui.MainStyle.BorderForeground(m.paneBorderColor(paneMain))
}

func (m model) renderBrowse(height, width int) string {
	innerH := height - 2
	indicator := "  "
	if m.focus == focusSearch {
		indicator = "> "
	}
	searchLine := indicator + "Search: " + m.query.View()
	listHeight := innerH - 2
	if listHeight < 1 {
		listHeight = 1
	}
	listView := m.list.View(width-2, listHeight, m.focus == focusList)
	if m.searching {
		listView = ui.MutedStyle.Render("Searching...")
	}
	content := searchLine + "\n\n" + listView
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(content)
}

func (m model) renderFavourites(height, width int) string {
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	listView := m.list.View(width-2, innerH, m.focus == focusList)
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(listView)
}

func (m model) renderPodcastSearch(height, width int) string {
	innerH := height - 2
	indicator := "  "
	if m.focus == focusSearch {
		indicator = "> "
	}
	searchLine := indicator + "Search: " + m.podcastQuery.View()
	listHeight := innerH - 2
	if listHeight < 1 {
		listHeight = 1
	}

	listView := m.podcastEpisodeList.View(width-2, listHeight, m.focus == focusList)
	if m.podcastSearching {
		listView = ui.MutedStyle.Render("Searching...")
	}
	content := searchLine + "\n\n" + listView
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(content)
}

func (m model) renderPodcastPreview(height, width int) string {
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	listView := m.podcastEpisodeList.View(width-2, innerH, m.focus == focusList)
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(listView)
}

func (m model) renderPodcastSubscriptions(height, width int) string {
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	listView := m.podcastEpisodeList.View(width-2, innerH, m.focus == focusList)
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(listView)
}

func (m model) renderPodcastEpisodes(height, width int) string {
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	listView := m.podcastEpisodeList.View(width-2, innerH, m.focus == focusList)
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(listView)
}

func (m model) renderPodcastNewEpisodes(height, width int) string {
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	listView := m.podcastEpisodeList.View(width-2, innerH, m.focus == focusList)
	if len(m.podcastEpisodeList.Items) == 0 {
		listView = ui.MutedStyle.Render("No new episodes.")
	}
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(listView)
}

func (m model) renderPodcastContinueListening(height, width int) string {
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}
	listView := m.podcastEpisodeList.View(width-2, innerH, m.focus == focusList)
	if len(m.podcastEpisodeList.Items) == 0 {
		listView = ui.MutedStyle.Render("No episodes in progress.")
	}
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(listView)
}

func (m model) settingsCount() int {
	return 7
}

func (m model) renderSettings(height, width int) string {
	labels := []string{
		fmt.Sprintf("MPV path:          %s", m.config.MpvPath),
		fmt.Sprintf("API host:          %s", m.config.APIHost),
		fmt.Sprintf("Default volume:    %d%%", m.config.Volume),
		fmt.Sprintf("Refresh interval:  %d min", m.config.PodcastRefreshInterval),
		fmt.Sprintf("Podcast browser:   %s", m.config.PodcastBrowser),
		fmt.Sprintf("Theme mode:        %s", m.config.ThemeMode),
		fmt.Sprintf("Theme:             %s", m.config.Theme),
	}
	lines := []string{ui.CategoryStyle.Render("Settings"), ""}
	for i, l := range labels {
		line := "  " + l
		if i == m.settingsCursor {
			line = ui.SelectedItemStyle.Render(line)
		}
		lines = append(lines, line)
	}
	return m.mainPanelStyle().Height(height).MaxHeight(height).Width(width - 2).Render(strings.Join(lines, "\n"))
}

func (m model) podcastBrowserOptions() []string {
	return []string{"itunes"}
}

func (m model) openSettingsEditor() (tea.Model, tea.Cmd) {
	m.settingInput.SetValue("")
	switch m.settingsCursor {
	case 0:
		m.settingInput.SetValue(m.config.MpvPath)
	case 1:
		m.settingInput.SetValue(m.config.APIHost)
	case 2:
		m.settingInput.SetValue(fmt.Sprintf("%d", m.config.Volume))
	case 3:
		m.settingInput.SetValue(fmt.Sprintf("%d", m.config.PodcastRefreshInterval))
	default:
		return m, nil
	}
	m.editingSetting = true
	m.settingInput.Focus()
	return m, textinput.Blink
}

func (m model) updateSettingsDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		return m.saveSettingValue()
	case tea.KeyEsc:
		m.editingSetting = false
		m.settingInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.settingInput, cmd = m.settingInput.Update(msg)
	return m, cmd
}

func (m model) cyclePodcastBrowser() (tea.Model, tea.Cmd) {
	opts := m.podcastBrowserOptions()
	if len(opts) == 0 {
		m.editingSetting = false
		return m, nil
	}
	idx := 0
	for i, o := range opts {
		if o == m.config.PodcastBrowser {
			idx = i
			break
		}
	}
	idx = (idx + 1) % len(opts)
	m.config.PodcastBrowser = opts[idx]
	m.editingSetting = false
	if err := store.SaveConfig(m.config); err != nil {
		m.errMsg = err.Error()
	}
	return m, nil
}

func (m model) cycleThemeMode() (tea.Model, tea.Cmd) {
	if m.config.ThemeMode == ui.ThemeModeDark {
		m.config.ThemeMode = ui.ThemeModeLight
	} else {
		m.config.ThemeMode = ui.ThemeModeDark
	}
	// Pick the first theme of the new mode so the UI updates immediately.
	t := ui.DefaultThemeForMode(m.config.ThemeMode)
	m.config.Theme = t.Name
	ui.SetTheme(t)
	m.editingSetting = false
	if err := store.SaveConfig(m.config); err != nil {
		m.errMsg = err.Error()
	}
	return m, nil
}

func (m model) cycleTheme() (tea.Model, tea.Cmd) {
	opts := ui.ThemeNamesByMode(m.config.ThemeMode)
	if len(opts) == 0 {
		m.editingSetting = false
		return m, nil
	}
	idx := 0
	for i, o := range opts {
		if o == m.config.Theme {
			idx = i
			break
		}
	}
	idx = (idx + 1) % len(opts)
	m.config.Theme = opts[idx]
	ui.SetTheme(ui.ThemeByName(m.config.Theme))
	m.editingSetting = false
	if err := store.SaveConfig(m.config); err != nil {
		m.errMsg = err.Error()
	}
	return m, nil
}

func (m model) saveSettingValue() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.settingInput.Value())
	switch m.settingsCursor {
	case 0:
		m.config.MpvPath = val
	case 1:
		m.config.APIHost = val
		m.client = radio.New(val)
	case 2:
		v, err := strconv.Atoi(val)
		if err != nil || v < 0 || v > 100 {
			m.errMsg = "Volume must be 0-100"
			return m, nil
		}
		m.config.Volume = v
		m.player.SetVolume(v)
	case 3:
		v, err := strconv.Atoi(val)
		if err != nil || v < 5 {
			m.errMsg = "Refresh interval must be at least 5 minutes"
			return m, nil
		}
		m.config.PodcastRefreshInterval = v
	}
	m.editingSetting = false
	m.settingInput.Blur()
	if err := store.SaveConfig(m.config); err != nil {
		m.errMsg = err.Error()
	}
	return m, nil
}

func (m model) renderSettingsDialog() string {
	titles := []string{
		"MPV path",
		"API host",
		"Default volume (0-100)",
		"Refresh interval (minutes, min 5)",
		"",
		"",
		"",
	}
	content := ui.CategoryStyle.Render(titles[m.settingsCursor]) + "\n\n" +
		m.settingInput.View() + "\n" +
		ui.MutedStyle.Render("Enter: save  •  Esc: cancel")

	box := lipgloss.NewStyle().
		Background(ui.ColorSurface()).
		Foreground(ui.ColorText()).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary()).
		Width(m.width - 20)
	return box.Render(content)
}

func (m model) hintText() string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.ColorPrimary()).Bold(true)
	descStyle := ui.MutedStyle

	kv := func(key, desc string) string {
		return keyStyle.Render(key+":") + " " + descStyle.Render(desc)
	}

	if m.focus == focusSidebar {
		return kv("↑↓", "navigate") + "  " + kv("→", "main panel") + "  " + kv("Enter", "select") + "  " + kv("?", "help") + "  " + kv("Q", "quit")
	}

	var parts []string
	switch m.view {
	case viewRadioBrowse:
		if m.focus == focusSearch {
			return kv("Enter", "search") + "  " + kv("Esc", "cancel")
		}
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "play"), kv("F", "save"), kv("/", "search"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewRadioFavourites:
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "play"), kv("F", "edit"), kv("D", "delete"), kv("C", "category"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewPodcastSearch:
		if m.focus == focusSearch {
			return kv("Enter", "search") + "  " + kv("Esc", "cancel")
		}
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "preview"), kv("A", "subscribe"), kv("/", "search"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewPodcastPreview:
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "play"), kv("A", "subscribe"), kv("Esc", "back"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewPodcastSubscriptions:
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "episodes"), kv("A", "add"), kv("D", "delete"), kv("R", "refresh"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewPodcastEpisodes:
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "play"), kv("R", "resume"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewPodcastNewEpisodes:
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "play"), kv("R", "refresh"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewPodcastContinueListening:
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "resume"), kv("R", "refresh"), kv("+", "vol up"), kv("-", "vol down"), kv("?", "help"), kv("Q", "quit")}
	case viewSettings:
		parts = []string{kv("↑↓", "navigate"), kv("Enter", "edit/cycle"), kv("?", "help"), kv("Q", "quit")}
	default:
		return kv("?", "help") + "  " + kv("Q", "quit")
	}
	return strings.Join(parts, "  ")
}

func (m model) renderHintBar(height int) string {
	content := m.hintText()
	if m.toast != "" {
		content = ui.CategoryStyle.Render(m.toast)
	}
	return lipgloss.NewStyle().
		Height(height).
		MaxHeight(height).
		Width(m.width).
		Render(content)
}

func (m model) renderSaveDialog() string {
	title := "Add to favourites"
	if m.saveOrigCategory != "" {
		title = "Edit favourite"
	}
	hints := ""
	if cats := m.favs.CategoryNames(); len(cats) > 0 {
		hints = ui.MutedStyle.Render("Existing: "+strings.Join(cats, ", ")) + "\n"
	}
	content := ui.CategoryStyle.Render(title) + "\n\n" +
		"Category\n" + m.saveCat.View() + "\n" +
		hints +
		ui.MutedStyle.Render("Tab: autocomplete  •  Enter: save  •  Esc: cancel")

	box := lipgloss.NewStyle().
		Background(ui.ColorSurface()).
		Foreground(ui.ColorText()).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary()).
		Width(m.width - 10)
	return box.Render(content)
}

func (m model) renderCreateCategoryDialog() string {
	content := ui.CategoryStyle.Render("Create category") + "\n\n" +
		m.catInput.View() + "\n\n" +
		ui.MutedStyle.Render("Enter: create  •  Esc: cancel")

	box := lipgloss.NewStyle().
		Background(ui.ColorSurface()).
		Foreground(ui.ColorText()).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary()).
		Width(m.width - 20)
	return box.Render(content)
}

func (m model) renderAddPodcastDialog() string {
	content := ui.CategoryStyle.Render("Add podcast feed") + "\n\n" +
		m.podcastAddInput.View() + "\n\n" +
		ui.MutedStyle.Render("Enter: fetch  •  Esc: cancel")

	box := lipgloss.NewStyle().
		Background(ui.ColorSurface()).
		Foreground(ui.ColorText()).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary()).
		Width(m.width - 20)
	return box.Render(content)
}

func (m model) renderResumePrompt() string {
	title := "Resume episode?"
	content := ui.CategoryStyle.Render(title) + "\n\n" +
		ui.MutedStyle.Render("R: resume  •  Enter/S: restart  •  Esc: cancel")

	box := lipgloss.NewStyle().
		Background(ui.ColorSurface()).
		Foreground(ui.ColorText()).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary()).
		Width(m.width - 20)
	return box.Render(content)
}

// --- Item builders ---

func (m model) buildStationItems() []ui.ListItem {
	items := make([]ui.ListItem, 0, len(m.stations))
	for _, s := range m.stations {
		items = append(items, ui.ListItem{
			Name:    s.Name,
			Codec:   strings.ToUpper(s.Codec),
			Bitrate: s.Bitrate,
			Country: s.CountryCode,
			Kind:    ui.KindStation,
		})
	}
	return items
}

func (m model) buildFavItems() []ui.ListItem {
	items := make([]ui.ListItem, 0)
	for _, cat := range m.favs.Categories {
		items = append(items, ui.ListItem{Name: cat.Name, IsHeader: true, Kind: ui.KindCategory})
		for _, fav := range cat.Stations {
			country := fav.CountryCode
			if country == "" {
				country = fav.Country
			}
			if country == "" {
				country = cat.Name
			}
			items = append(items, ui.ListItem{
				Name:    fav.Name,
				Codec:   strings.ToUpper(fav.Codec),
				Bitrate: fav.Bitrate,
				Country: country,
				Kind:    ui.KindFavourite,
			})
		}
	}
	return items
}

// selectedFavCategory returns the nearest category header above the current cursor.
func (m model) selectedFavCategory() string {
	for i := m.list.Cursor; i >= 0; i-- {
		if m.list.Items[i].IsHeader {
			return m.list.Items[i].Name
		}
	}
	return ""
}

func (m model) buildPodcastSearchItems() []ui.ListItem {
	items := make([]ui.ListItem, 0, len(m.podcastSearchResults))
	for _, r := range m.podcastSearchResults {
		items = append(items, ui.ListItem{
			Name: r.Title,
			Kind: ui.KindPodcastFeed,
		})
	}
	return items
}

func (m model) buildSubscriptionItems() []ui.ListItem {
	items := make([]ui.ListItem, 0, len(m.podcasts))
	for _, f := range m.podcasts {
		items = append(items, ui.ListItem{
			Name: f.Title,
			Kind: ui.KindPodcastFeed,
		})
	}
	return items
}

func (m model) buildPodcastEpisodeItems(feed *podcast.Feed) []ui.ListItem {
	items := make([]ui.ListItem, 0, len(feed.Episodes))
	for _, ep := range feed.Episodes {
		status := " "
		switch {
		case ep.Completed:
			status = "✓"
		case ep.Played:
			status = "○"
		default:
			status = "●"
		}
		items = append(items, ui.ListItem{
			Name:     ep.Title,
			Duration: formatDuration(ep.Duration),
			Date:     formatDate(ep.PubDate),
			Status:   status,
			FeedURL:  feed.URL,
			URL:      ep.URL,
			Kind:     ui.KindPodcastEpisode,
		})
	}
	return items
}

func (m model) buildPreviewItems(feed *podcast.Feed, width int) []ui.ListItem {
	items := make([]ui.ListItem, 0)

	// Title as a header so it is rendered prominently and skipped on navigation.
	items = append(items, ui.ListItem{
		Name:     "Preview: " + feed.Title,
		IsHeader: true,
	})

	// Blank line after title.
	items = append(items, ui.ListItem{Name: "", Kind: ui.KindPreviewInfo})

	// Full description, word-wrapped to the available width.
	for _, line := range wrapLines(feed.Description, width) {
		items = append(items, ui.ListItem{
			Name: line,
			Kind: ui.KindPreviewInfo,
		})
	}

	// Metadata line.
	updated := formatDate(feed.LastFetched)
	if updated == "?" {
		updated = "never"
	}
	epCount := len(feed.Episodes)
	meta := fmt.Sprintf("Latest %d of %d episodes", min(epCount, 10), epCount)
	if len(feed.Categories) > 0 {
		meta += " · " + strings.Join(feed.Categories, ", ")
	}
	meta += " · Updated: " + updated
	items = append(items, ui.ListItem{
		Name: meta,
		Kind: ui.KindPreviewMeta,
	})

	// Separator between metadata and episodes.
	sep := strings.Repeat("─", max(width, 1))
	items = append(items, ui.ListItem{Name: sep, Kind: ui.KindPreviewMeta})

	// Latest 10 episodes, newest first.
	eps := make([]podcast.Episode, len(feed.Episodes))
	copy(eps, feed.Episodes)
	sort.Slice(eps, func(i, j int) bool {
		return eps[i].PubDate.After(eps[j].PubDate)
	})
	if len(eps) > 10 {
		eps = eps[:10]
	}
	for i := range eps {
		ep := &eps[i]
		items = append(items, ui.ListItem{
			Name:     ep.Title,
			Duration: formatDuration(ep.Duration),
			Date:     formatDate(ep.PubDate),
			Status:   "●",
			FeedURL:  feed.URL,
			URL:      ep.URL,
			Kind:     ui.KindPodcastEpisode,
		})
	}
	return items
}

func (m model) buildNewEpisodeItems() []ui.ListItem {
	items := make([]ui.ListItem, 0)
	for i := range m.podcasts {
		feed := &m.podcasts[i]
		for _, ep := range feed.Episodes {
			if !ep.IsNew {
				continue
			}
			items = append(items, ui.ListItem{
				Name:     feed.Title + " — " + ep.Title,
				Duration: formatDuration(ep.Duration),
				Date:     formatDate(ep.PubDate),
				Status:   "●",
				FeedURL:  feed.URL,
				URL:      ep.URL,
				Kind:     ui.KindPodcastEpisode,
			})
		}
	}
	sortPodcastItemsByDate(items)
	return items
}

func (m model) buildContinueListeningItems() []ui.ListItem {
	items := make([]ui.ListItem, 0)
	for i := range m.podcasts {
		feed := &m.podcasts[i]
		for _, ep := range feed.Episodes {
			if ep.Position <= 0 || ep.Completed {
				continue
			}
			items = append(items, ui.ListItem{
				Name:     feed.Title + " — " + ep.Title,
				Duration: formatDuration(ep.Duration),
				Date:     formatDuration(ep.Position) + " / " + formatDuration(ep.Duration),
				Status:   "○",
				FeedURL:  feed.URL,
				URL:      ep.URL,
				Kind:     ui.KindPodcastEpisode,
			})
		}
	}
	sortPodcastItemsByDate(items)
	return items
}

func sortPodcastItemsByDate(items []ui.ListItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Date > items[j].Date
	})
}

// --- Utilities ---

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "?"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func (m model) renderProgressBar(position, duration, width int) string {
	if duration <= 0 {
		return strings.Repeat("░", width)
	}
	filled := int(float64(position) / float64(duration) * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	empty := width - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	percent := int(float64(position) / float64(duration) * 100)
	return fmt.Sprintf("%s %3d%%", bar, percent)
}

func renderInitials(width int, name string) string {
	initials := "?"
	if name != "" {
		parts := strings.Fields(name)
		if len(parts) >= 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
			initials = strings.ToUpper(string(parts[0][0]) + string(parts[1][0]))
		} else if len(name) >= 2 {
			initials = strings.ToUpper(name[:2])
		} else {
			initials = strings.ToUpper(name)
		}
	}

	fig := figure.NewFigure(initials, "standard", true)
	art := strings.TrimRight(fig.String(), "\n")
	lines := strings.Split(art, "\n")

	centerStyle := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Background(ui.ColorBg()).
		Foreground(ui.ColorPrimary()).
		Bold(true)

	maxW := 0
	for _, line := range lines {
		if w := runewidth.StringWidth(line); w > maxW {
			maxW = w
		}
	}

	var rendered []string
	for _, line := range lines {
		padded := line
		if w := runewidth.StringWidth(line); w < maxW {
			padded += strings.Repeat(" ", maxW-w)
		}
		rendered = append(rendered, centerStyle.Render(runewidth.Truncate(padded, width, "")))
	}
	return strings.Join(rendered, "\n")
}

func formatTags(tags string) string {
	if tags == "" {
		return ""
	}
	parts := strings.Split(tags, ",")
	var out []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, "#"+t)
		}
	}
	return strings.Join(out, " ")
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	return t.Format("2006-01-02")
}

// wrapLines word-wraps s into lines of at most width display cells. Existing
// newlines are preserved as paragraph breaks, and words longer than width are
// hard-truncated so nothing is silently lost at the right edge.
func wrapLines(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}

	var lines []string
	for _, para := range strings.Split(s, "\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(para)
		var b strings.Builder
		currentW := 0

		for _, w := range words {
			ww := runewidth.StringWidth(w)
			// Hard-truncate a word that cannot fit on a line by itself.
			if ww > width {
				if b.Len() > 0 {
					lines = append(lines, b.String())
					b.Reset()
					currentW = 0
				}
				lines = append(lines, runewidth.Truncate(w, width, ""))
				continue
			}

			space := 0
			if b.Len() > 0 {
				space = 1
			}
			if currentW+space+ww > width {
				lines = append(lines, b.String())
				b.Reset()
				currentW = 0
				space = 0
			}
			if b.Len() > 0 {
				b.WriteByte(' ')
				currentW++
			}
			b.WriteString(w)
			currentW += ww
		}
		if b.Len() > 0 {
			lines = append(lines, b.String())
		}
	}
	return lines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mergeEpisodes(fetched []podcast.Episode, existing []podcast.Episode) []podcast.Episode {
	state := make(map[string]podcast.Episode)
	for _, e := range existing {
		state[e.URL] = e
	}
	merged := make([]podcast.Episode, 0, len(fetched))
	for _, e := range fetched {
		if old, ok := state[e.URL]; ok {
			e.Position = old.Position
			e.Played = old.Played
			e.Completed = old.Completed
			e.IsNew = old.IsNew
		} else {
			e.IsNew = true
		}
		merged = append(merged, e)
	}
	return merged
}
