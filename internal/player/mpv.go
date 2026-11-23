package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Media describes anything that can be played (station, podcast episode, etc.).
type Media struct {
	Name       string
	URL        string
	Codec      string
	Bitrate    int
	Country    string
	Tags       string
	ResumeFrom int // seconds to resume from (0 = start from beginning)
	IsPodcast  bool
}

// StatusMsg is sent back into the Bubble Tea program whenever playback state changes.
type StatusMsg struct {
	Playing  bool
	Paused   bool
	Media    Media
	Title    string
	Codec    string
	Bitrate  int
	Volume   int
	Position int // current playback position in seconds
	Error    string
}

// FinishedMsg is sent when the current file reaches end-of-file.
type FinishedMsg struct{}

// MPV wraps an mpv process controlled via JSON IPC.
type MPV struct {
	events  chan<- interface{}
	mpvPath string
	socket  string
	proc    *os.Process
	conn    net.Conn
	mu      sync.Mutex

	status        StatusMsg
	pendingResume float64
}

// New creates a new mpv-backed player. Events are pushed on the supplied channel.
func New(events chan<- interface{}, mpvPath, socket string) *MPV {
	if mpvPath == "" {
		mpvPath = "mpv"
	}
	if socket == "" {
		socket = filepath.Join(os.TempDir(), "zuelose-mpv.sock")
	}
	return &MPV{
		events:  events,
		mpvPath: mpvPath,
		socket:  socket,
	}
}

// Socket returns the IPC socket path.
func (m *MPV) Socket() string { return m.socket }

// Start launches mpv and connects to its IPC socket.
func (m *MPV) Start() error {
	_ = os.Remove(m.socket)

	cmd := exec.Command(m.mpvPath,
		"--no-video",
		"--no-terminal",
		"--idle",
		"--input-ipc-server="+m.socket,
	)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mpv: %w", err)
	}
	m.proc = cmd.Process

	// Wait for the socket to appear.
	found := false
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(m.socket); err == nil {
			found = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		_ = m.proc.Kill()
		return fmt.Errorf("mpv IPC socket did not appear")
	}

	conn, err := net.Dial("unix", m.socket)
	if err != nil {
		_ = m.proc.Kill()
		return fmt.Errorf("failed to connect to mpv socket: %w", err)
	}
	m.conn = conn

	// Observe useful properties.
	observations := [][]interface{}{
		{"observe_property", 1, "metadata"},
		{"observe_property", 2, "audio-codec-name"},
		{"observe_property", 3, "audio-bitrate"},
		{"observe_property", 4, "volume"},
		{"observe_property", 5, "pause"},
		{"observe_property", 6, "idle-active"},
		{"observe_property", 7, "time-pos"},
	}
	for _, o := range observations {
		if err := m.writeCommand(o...); err != nil {
			_ = m.Close()
			return err
		}
	}

	go m.reader()
	return nil
}

// Close shuts down mpv and cleans up the socket.
func (m *MPV) Close() error {
	if m.conn != nil {
		_ = m.conn.Close()
	}
	if m.proc != nil {
		_ = m.proc.Kill()
	}
	_ = os.Remove(m.socket)
	return nil
}

// Play starts playback of the given media.
func (m *MPV) Play(media Media) {
	m.mu.Lock()
	m.status.Media = media
	m.status.Codec = media.Codec
	m.status.Bitrate = media.Bitrate
	m.status.Title = ""
	m.status.Playing = true
	m.status.Paused = false
	m.status.Position = 0
	m.pendingResume = 0
	if media.ResumeFrom > 0 {
		m.pendingResume = math.Max(0, float64(media.ResumeFrom-1))
	}
	m.mu.Unlock()
	_ = m.writeCommand("loadfile", media.URL)
	m.sendStatus()
}

// Stop halts playback.
func (m *MPV) Stop() {
	_ = m.writeCommand("stop")
}

// TogglePause pauses or resumes playback.
func (m *MPV) TogglePause() {
	m.mu.Lock()
	paused := !m.status.Paused
	m.mu.Unlock()
	_ = m.writeCommand("set_property", "pause", paused)
}

// VolumeUp increases volume by 5, clamped at 100.
func (m *MPV) VolumeUp() {
	m.mu.Lock()
	vol := m.status.Volume + 5
	if vol > 100 {
		vol = 100
	}
	m.mu.Unlock()
	_ = m.writeCommand("set_property", "volume", vol)
}

// VolumeDown decreases volume by 5, clamped at 0.
func (m *MPV) VolumeDown() {
	m.mu.Lock()
	vol := m.status.Volume - 5
	if vol < 0 {
		vol = 0
	}
	m.mu.Unlock()
	_ = m.writeCommand("set_property", "volume", vol)
}

// SetVolume sets an absolute volume level.
func (m *MPV) SetVolume(vol int) {
	if vol < 0 {
		vol = 0
	}
	if vol > 100 {
		vol = 100
	}
	_ = m.writeCommand("set_property", "volume", vol)
}

// Status returns the current playback status snapshot.
func (m *MPV) Status() StatusMsg {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

func (m *MPV) writeCommand(args ...interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.conn == nil {
		return fmt.Errorf("not connected to mpv")
	}
	req := map[string]interface{}{"command": args}
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = m.conn.Write(b)
	return err
}

func (m *MPV) reader() {
	scanner := bufio.NewScanner(m.conn)
	for scanner.Scan() {
		line := scanner.Text()
		m.handleLine(line)
	}
}

func (m *MPV) handleLine(line string) {
	var evt map[string]interface{}
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		return
	}

	ev, _ := evt["event"].(string)
	if ev == "file-loaded" {
		m.mu.Lock()
		resume := m.pendingResume
		m.pendingResume = 0
		m.mu.Unlock()
		if resume > 0 {
			_ = m.writeCommand("set_property", "time-pos", resume)
		}
		return
	}

	if ev == "end-file" {
		reason, _ := evt["reason"].(string)
		if reason == "eof" {
			m.sendFinished()
		}
		return
	}

	if ev != "property-change" {
		return
	}

	idf, _ := evt["id"].(float64)
	id := int(idf)
	data := evt["data"]

	m.mu.Lock()
	switch id {
	case 1: // metadata
		title := extractTitle(data)
		if title != "" {
			m.status.Title = title
		}
	case 2: // audio-codec-name
		if s, ok := data.(string); ok && s != "" {
			m.status.Codec = strings.ToUpper(s)
		}
	case 3: // audio-bitrate
		if f, ok := data.(float64); ok && f > 0 {
			// mpv may report bps or kbps depending on version/property.
			bitrate := int(f)
			if bitrate > 1000000 {
				bitrate = bitrate / 1000
			}
			m.status.Bitrate = bitrate
		}
	case 4: // volume
		if f, ok := data.(float64); ok {
			m.status.Volume = int(f)
		}
	case 5: // pause
		if b, ok := data.(bool); ok {
			m.status.Paused = b
			m.status.Playing = !b
		}
	case 6: // idle-active
		if b, ok := data.(bool); ok && b {
			m.status.Playing = false
			m.status.Paused = false
			m.status.Title = ""
			m.status.Position = 0
		}
	case 7: // time-pos
		if f, ok := data.(float64); ok && f >= 0 {
			m.status.Position = int(f)
		}
	}
	m.mu.Unlock()

	m.sendStatus()
}

func (m *MPV) sendStatus() {
	m.mu.Lock()
	msg := m.status
	m.mu.Unlock()
	if m.events != nil {
		select {
		case m.events <- msg:
		default:
		}
	}
}

func (m *MPV) sendFinished() {
	if m.events != nil {
		select {
		case m.events <- FinishedMsg{}:
		default:
		}
	}
}

func extractTitle(data interface{}) string {
	m, ok := data.(map[string]interface{})
	if !ok {
		return ""
	}

	// Try explicit title keys first.
	for _, key := range []string{"title", "TITLE", "icy-title", "ICY-TITLE"} {
		if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}

	// Fall back to any key that looks like a title.
	for k, v := range m {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "title") {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// FormatBitrate returns a human-readable bitrate string.
func FormatBitrate(kbps int) string {
	if kbps <= 0 {
		return "?"
	}
	return strconv.Itoa(kbps) + "k"
}
