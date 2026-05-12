// Package ws implements a minimal websocket fan-out hub for pr-watcher events.
//
// Slow consumers are dropped after a 2s send timeout to keep the hub non-blocking.
package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const sendTimeout = 2 * time.Second

// Event is the on-the-wire payload. Callers fill in whichever fields are relevant.
type Event struct {
	Type      string    `json:"type"`
	Provider  string    `json:"provider,omitempty"`
	Repo      string    `json:"repo,omitempty"`
	Number    int       `json:"number,omitempty"`
	HeadSHA   string    `json:"head_sha,omitempty"`
	Title     string    `json:"title,omitempty"`
	URL       string    `json:"url,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
	// Free-form extra payload for review_started/posted body, log lines, etc.
	Detail string `json:"detail,omitempty"`
}

// Hub owns the set of connected websocket clients.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
	log     *slog.Logger
}

type client struct {
	conn *websocket.Conn
	send chan []byte
}

// New returns an empty Hub.
func New(log *slog.Logger) *Hub {
	return &Hub{clients: map[*client]struct{}{}, log: log}
}

// ClientCount returns the current number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// Publish marshals evt and broadcasts to every client with a 2s timeout per send.
func (h *Hub) Publish(evt Event) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	b, err := json.Marshal(evt)
	if err != nil {
		if h.log != nil {
			h.log.Warn("ws marshal failed", "err", err.Error())
		}
		return
	}
	h.broadcast(b)
}

// broadcast is exposed for tests via BroadcastRaw.
func (h *Hub) broadcast(payload []byte) {
	h.mu.Lock()
	targets := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.Unlock()
	for _, c := range targets {
		select {
		case c.send <- payload:
		default:
			// Buffer full — try with a bounded wait, then drop.
			go h.deliverOrDrop(c, payload)
		}
	}
}

// BroadcastRaw is exported for tests that want to push opaque bytes.
func (h *Hub) BroadcastRaw(payload []byte) { h.broadcast(payload) }

func (h *Hub) deliverOrDrop(c *client, payload []byte) {
	t := time.NewTimer(sendTimeout)
	defer t.Stop()
	select {
	case c.send <- payload:
	case <-t.C:
		h.remove(c)
		_ = c.conn.Close(websocket.StatusPolicyViolation, "slow consumer")
	}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

// ServeHTTP handles the websocket upgrade. Mount it at e.g. "/ws".
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // browsers from local UI; cross-origin is fine here.
	})
	if err != nil {
		if h.log != nil {
			h.log.Warn("ws accept failed", "err", err.Error())
		}
		return
	}
	c := &client{conn: conn, send: make(chan []byte, 64)}
	h.add(c)
	if h.log != nil {
		h.log.Info("ws client connected", "remote", r.RemoteAddr, "clients", h.ClientCount())
	}
	ctx := conn.CloseRead(r.Context())
	defer func() {
		h.remove(c)
		_ = conn.Close(websocket.StatusNormalClosure, "bye")
		if h.log != nil {
			h.log.Info("ws client disconnected", "remote", r.RemoteAddr, "clients", h.ClientCount())
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, sendTimeout)
			err := conn.Write(writeCtx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// Shutdown closes all clients with a normal-closure status.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	for c := range h.clients {
		_ = c.conn.Close(websocket.StatusNormalClosure, "shutdown")
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}
