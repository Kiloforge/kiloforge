package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"kiloforge/internal/core/domain"
)

// SSEHub manages SSE client connections and broadcasts events.
// It implements port.EventBus.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[chan domain.Event]struct{}
}

// NewSSEHub creates a new hub.
func NewSSEHub() *SSEHub {
	return &SSEHub{
		clients: make(map[chan domain.Event]struct{}),
	}
}

// Subscribe registers a new client and returns its event channel.
func (h *SSEHub) Subscribe() <-chan domain.Event {
	ch := make(chan domain.Event, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a client and closes its channel.
func (h *SSEHub) Unsubscribe(ch <-chan domain.Event) {
	// We need the bidirectional channel to delete from the map.
	// The subscribe method creates a bidirectional channel and returns
	// a receive-only view. We recover the original via type assertion
	// on the map lookup. This is safe because only Subscribe creates channels.
	h.mu.Lock()
	for bch := range h.clients {
		if (<-chan domain.Event)(bch) == ch {
			delete(h.clients, bch)
			h.mu.Unlock()
			close(bch)
			return
		}
	}
	h.mu.Unlock()
}

// Publish sends an event to all connected clients.
// Slow clients that can't keep up will miss events (non-blocking send).
func (h *SSEHub) Publish(event domain.Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

// ClientCount returns the number of connected SSE clients.
func (h *SSEHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	ch := s.hub.Subscribe()
	defer s.hub.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()
		}
	}
}
