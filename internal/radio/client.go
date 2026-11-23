package radio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/daum3ns/zuelose/internal/store"
)

// Client queries RadioBrowser.
type Client struct {
	Host   string
	Scheme string
	client *http.Client
}

// New creates a new RadioBrowser client.
func New(host string) *Client {
	if host == "" {
		host = "all.api.radio-browser.info"
	}
	return &Client{
		Host:   host,
		Scheme: "https",
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Search searches stations by name. limit <= 0 defaults to 50.
func (c *Client) Search(ctx context.Context, name string, limit int) ([]store.Station, error) {
	if limit <= 0 {
		limit = 50
	}
	scheme := c.Scheme
	if scheme == "" {
		scheme = "https"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   c.Host,
		Path:   "/json/stations/search",
	}
	q := u.Query()
	if name != "" {
		q.Set("name", name)
	}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("order", "clickcount")
	q.Set("reverse", "true")
	q.Set("hidebroken", "true")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "zuelose/0.1")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("radio browser returned %s", resp.Status)
	}

	var stations []store.Station
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, err
	}
	return stations, nil
}
