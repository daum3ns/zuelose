package radio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func hostFromURL(t *testing.T, raw string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u.Host
}

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/stations/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"name": "Station One", "url": "http://one", "codec": "MP3", "bitrate": 128, "countrycode": "US"},
			{"name": "Station Two", "url": "http://two", "codec": "AAC", "bitrate": 256, "countrycode": "CH"}
		]`))
	}))
	defer server.Close()

	client := New(hostFromURL(t, server.URL))
	client.Scheme = "http"
	client.client = server.Client()
	stations, err := client.Search(context.Background(), "test", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stations) != 2 {
		t.Fatalf("expected 2 stations, got %d", len(stations))
	}
	if stations[0].Name != "Station One" {
		t.Fatalf("unexpected name: %q", stations[0].Name)
	}
}

func TestSearchServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client := New(hostFromURL(t, server.URL))
	client.Scheme = "http"
	client.client = server.Client()
	_, err := client.Search(context.Background(), "test", 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewDefaults(t *testing.T) {
	c := New("")
	if c.Host != "all.api.radio-browser.info" {
		t.Fatalf("expected default host, got %q", c.Host)
	}
}
