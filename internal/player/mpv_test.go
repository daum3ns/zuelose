package player

import (
	"bufio"
	"encoding/json"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestPlayUnpausesLoadedMedia(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	m := &MPV{
		conn:   client,
		events: make(chan interface{}, 1),
		done:   make(chan struct{}),
	}
	m.status.Media = Media{Name: "Podcast", URL: "http://episode", IsPodcast: true}
	m.status.Playing = false
	m.status.Paused = true

	commands := make(chan []interface{}, 2)
	readErr := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(server)
		for scanner.Scan() {
			var req struct {
				Command []interface{} `json:"command"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				readErr <- err
				return
			}
			commands <- req.Command
		}
		readErr <- scanner.Err()
	}()

	m.Play(Media{Name: "Radio", URL: "http://station"})

	want := [][]interface{}{
		{"loadfile", "http://station"},
		{"set_property", "pause", false},
	}
	for i, expected := range want {
		select {
		case got := <-commands:
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("command %d = %#v, want %#v", i, got, expected)
			}
		case err := <-readErr:
			t.Fatalf("reading mpv commands: %v", err)
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for command %d", i)
		}
	}

	status := m.Status()
	if status.Paused || !status.Playing {
		t.Fatalf("expected new media to be playing, got %+v", status)
	}
}
