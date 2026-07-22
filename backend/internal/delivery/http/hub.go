package http

import (
	"sync"

	"github.com/gorilla/websocket"
)

// wsEnvelope is the single frame shape sent over every websocket connection.
// Type lets the client dispatch to the right handler (chat vs notification)
// without needing separate sockets per feature.
type wsEnvelope struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// connEntry pairs a connection with a mutex so concurrent Send calls to the
// same user (e.g. a chat_message dispatch racing a notification push that
// happens to land at the same moment) serialize instead of both calling
// gorilla/websocket's WriteJSON at once — which gorilla explicitly does not
// allow (one concurrent writer per connection) and panics the whole process
// if violated.
type connEntry struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// Hub tracks one live websocket connection per user and lets any part of the
// backend push a real-time frame to that user if they're online. It's the
// single seam other packages (chat delivery, notification creation) push
// through, so features get realtime delivery without owning connection
// lifecycle themselves. All writes to a given connection — including a
// handler's own echo/error frames on the connection it's holding — must go
// through Send so they share connEntry's write lock.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*connEntry
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]*connEntry)}
}

func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, exists := h.clients[userID]; exists {
		_ = old.conn.Close()
	}
	h.clients[userID] = &connEntry{conn: conn}
}

func (h *Hub) Deregister(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if entry, exists := h.clients[userID]; exists && entry.conn == conn {
		delete(h.clients, userID)
	}
}

// IsOnline reports whether userID currently has a live websocket connection.
// Satisfies usecase.PresenceChecker structurally — no import of that
// package needed here.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, online := h.clients[userID]
	return online
}

// Send pushes a typed frame to userID's live connection, if any. It's a
// no-op (returns false) when the user isn't currently connected. Safe to
// call concurrently for the same user from multiple goroutines.
func (h *Hub) Send(userID string, msgType string, data interface{}) bool {
	h.mu.RLock()
	entry, online := h.clients[userID]
	h.mu.RUnlock()
	if !online {
		return false
	}
	go func() {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		_ = entry.conn.WriteJSON(wsEnvelope{Type: msgType, Data: data})
	}()
	return true
}

// SendToMany pushes the same frame to several users at once — the seam a
// multi-party room (e.g. a dispute chat between two parties and a
// moderator) fans a message out through, reusing the exact same per-
// connection locking as Send so it composes safely with it.
func (h *Hub) SendToMany(userIDs []string, msgType string, data interface{}) {
	for _, id := range userIDs {
		if id != "" {
			h.Send(id, msgType, data)
		}
	}
}
