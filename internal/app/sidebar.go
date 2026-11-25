package app

// view identifies the active screen.
type view string

const (
	viewRadioBrowse              view = "radio-browse"
	viewRadioFavourites          view = "radio-favourites"
	viewPodcastSearch            view = "podcast-search"
	viewPodcastPreview           view = "podcast-preview"
	viewPodcastSubscriptions     view = "podcast-subscriptions"
	viewPodcastEpisodes          view = "podcast-episodes"
	viewPodcastNewEpisodes       view = "podcast-new-episodes"
	viewPodcastContinueListening view = "podcast-continue-listening"
	viewSettings                 view = "settings"
)

// sidebarItem is one entry in the flat navigation list.
type sidebarItem struct {
	label    string
	v        view
	isHeader bool
	depth    int
}

// newSidebar returns the flat navigation list.
func newSidebar() []sidebarItem {
	return []sidebarItem{
		{label: "Radio", isHeader: true, depth: 0},
		{label: "Browse", v: viewRadioBrowse, depth: 1},
		{label: "Favourites", v: viewRadioFavourites, depth: 1},
		{label: "Podcasts", isHeader: true, depth: 0},
		{label: "Search", v: viewPodcastSearch, depth: 1},
		{label: "Subscriptions", v: viewPodcastSubscriptions, depth: 1},
		{label: "New Episodes", v: viewPodcastNewEpisodes, depth: 1},
		{label: "Continue Listening", v: viewPodcastContinueListening, depth: 1},
		{label: "Settings", v: viewSettings, depth: 0},
	}
}

// findSidebarIndex returns the index of the first item matching the view.
func findSidebarIndex(items []sidebarItem, v view) int {
	for i, it := range items {
		if it.v == v {
			return i
		}
	}
	return -1
}
