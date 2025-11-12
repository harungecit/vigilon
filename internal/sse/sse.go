package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Client represents an SSE client connection
type Client struct {
	ID      string
	Channel chan []byte
	Context context.Context
}

// Manager manages SSE connections
type Manager struct {
	clients     map[string]*Client
	register    chan *Client
	unregister  chan *Client
	broadcast   chan []byte
	mutex       sync.RWMutex
	broadcaster func(ctx context.Context)
}

// NewManager creates a new SSE manager
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

// Start starts the SSE manager
func (m *Manager) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case client := <-m.register:
				m.mutex.Lock()
				m.clients[client.ID] = client
				m.mutex.Unlock()
			case client := <-m.unregister:
				m.mutex.Lock()
				if _, ok := m.clients[client.ID]; ok {
					close(client.Channel)
					delete(m.clients, client.ID)
				}
				m.mutex.Unlock()
			case message := <-m.broadcast:
				m.mutex.RLock()
				for _, client := range m.clients {
					select {
					case client.Channel <- message:
					case <-time.After(1 * time.Second):
						// Skip slow clients
					}
				}
				m.mutex.RUnlock()
			}
		}
	}()

	// Start broadcaster if configured
	if m.broadcaster != nil {
		go m.broadcaster(ctx)
	}
}

// ServeHTTP handles SSE connections
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Create client
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	client := &Client{
		ID:      clientID,
		Channel: make(chan []byte, 10),
		Context: r.Context(),
	}

	// Register client
	m.register <- client

	// Deregister on close
	defer func() {
		m.unregister <- client
	}()

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"clientId\":\"%s\"}\n\n", clientID)
	flusher.Flush()

	// Keepalive ticker to prevent timeout
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	// Stream messages
	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepalive.C:
			// Send keepalive comment (ignored by EventSource)
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case message, ok := <-client.Channel:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", message)
			flusher.Flush()
		}
	}
}

// Broadcast sends a message to all connected clients
func (m *Manager) Broadcast(eventType string, data interface{}) error {
	message := map[string]interface{}{
		"type": eventType,
		"data": data,
		"time": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	select {
	case m.broadcast <- jsonData:
	case <-time.After(100 * time.Millisecond):
		// Channel full, drop message
	}

	return nil
}

// SetBroadcaster sets a custom broadcaster function
func (m *Manager) SetBroadcaster(fn func(ctx context.Context)) {
	m.broadcaster = fn
}

// ClientCount returns the number of connected clients
func (m *Manager) ClientCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.clients)
}
