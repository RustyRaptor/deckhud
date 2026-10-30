// Package ipc pushes HUD events to overlay clients as JSON text frames over a
// loopback WebSocket. QML can speak WebSocket natively (QtWebSockets), which
// keeps the UI free of compiled code. Clients must know a random token, which
// the daemon hands to the UI it launches.
package ipc

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

// queueSize bounds how far a client may lag before frames are dropped. The HUD
// only cares about recent events, so dropping beats blocking the engine.
const queueSize = 256

// Hub fans events out to every connected client.
type Hub struct {
	token string
	greet func() []any

	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

// NewHub serves clients at "/<token>". greet returns the messages a new client
// receives first (e.g. current action set).
func NewHub(token string, greet func() []any) *Hub {
	return &Hub{token: token, greet: greet, clients: map[chan []byte]struct{}{}}
}

// Clients returns the number of connected clients.
func (h *Hub) Clients() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// Broadcast sends v to all clients without blocking.
func (h *Hub) Broadcast(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for q := range h.clients {
		select {
		case q <- data:
		default: // client is lagging; drop
		}
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/"+h.token {
		http.NotFound(w, r)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx := conn.CloseRead(r.Context()) // we never expect client messages

	q := make(chan []byte, queueSize)
	if h.greet != nil {
		for _, m := range h.greet() {
			if data, err := json.Marshal(m); err == nil {
				q <- data
			}
		}
	}
	h.mu.Lock()
	h.clients[q] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.clients, q)
		h.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case data := <-q:
			if err := write(ctx, conn, data); err != nil {
				return
			}
		}
	}
}

func write(ctx context.Context, c *websocket.Conn, data []byte) error {
	return c.Write(ctx, websocket.MessageText, data)
}
