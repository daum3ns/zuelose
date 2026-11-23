package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/daum3ns/zuelose/internal/app"
	"github.com/daum3ns/zuelose/internal/player"
	"github.com/daum3ns/zuelose/internal/radio"
	"github.com/daum3ns/zuelose/internal/store"
)

func main() {
	cfg, err := store.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load config: %v\n", err)
		cfg = store.DefaultConfig()
	}

	favs, err := store.LoadFavourites()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not load favourites: %v\n", err)
	}

	statusCh := make(chan interface{}, 16)
	mpv := player.New(statusCh, cfg.MpvPath, "")
	client := radio.New(cfg.APIHost)

	m := app.NewModel(cfg, favs, mpv, client)
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Forward player events into the Bubble Tea program.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for msg := range statusCh {
			p.Send(msg)
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	_ = mpv.Close()
	close(statusCh)
	<-done
}
