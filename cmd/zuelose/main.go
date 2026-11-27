package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/daum3ns/zuelose/internal/app"
	"github.com/daum3ns/zuelose/internal/player"
	"github.com/daum3ns/zuelose/internal/radio"
	"github.com/daum3ns/zuelose/internal/store"
	"github.com/daum3ns/zuelose/internal/version"
)

func main() {
	if handleVersionArgs(os.Args[1:], os.Stdout) {
		return
	}
	run()
}

func handleVersionArgs(args []string, out io.Writer) bool {
	if len(args) != 1 {
		return false
	}
	switch args[0] {
	case "version", "--version", "-version":
		fmt.Fprintf(out, "zuelose version %s\n", version.String())
		return true
	default:
		return false
	}
}

func run() {
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
