package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"

	"apex/api-gateway/internal/app"
)

// Hub manages connected WebSocket clients and broadcasts decision events.
// It implements app.EventBus.
type Hub struct {
	clients    map[*websocket.Conn]struct{}
	mu         sync.RWMutex
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan app.DecisionEvent
}

// NewHub creates a Hub ready to be started.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]struct{}),
		register:   make(chan *websocket.Conn, 16),
		unregister: make(chan *websocket.Conn, 16),
		broadcast:  make(chan app.DecisionEvent, 64),
	}
}

// Run processes register/unregister/broadcast events. Call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = struct{}{}
			h.mu.Unlock()

		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
			}
			h.mu.Unlock()

		case event := <-h.broadcast:
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("ws hub: marshal event: %v", err)
				continue
			}
			h.mu.RLock()
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("ws hub: write to client: %v", err)
					// Schedule removal without holding the read lock.
					go func(c *websocket.Conn) { h.unregister <- c }(conn)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Register adds a connection to the hub.
func (h *Hub) Register(conn *websocket.Conn) {
	h.register <- conn
}

// Unregister removes a connection from the hub and closes it.
func (h *Hub) Unregister(conn *websocket.Conn) {
	h.unregister <- conn
}

// Broadcast sends a DecisionEvent to all connected clients. Implements app.EventBus.
func (h *Hub) Broadcast(event app.DecisionEvent) {
	h.broadcast <- event
}
