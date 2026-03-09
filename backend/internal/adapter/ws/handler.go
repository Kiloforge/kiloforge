package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"nhooyr.io/websocket"
)

// Handler handles WebSocket upgrade requests for interactive agent sessions.
type Handler struct {
	sessions *SessionManager
	logger   *log.Logger
}

// NewHandler creates a new WebSocket handler.
func NewHandler(sessions *SessionManager, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{sessions: sessions, logger: logger}
}

// RegisterRoutes registers the WebSocket endpoint on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ws/agent/{id}", h.handleAgentWS)
}

// handleAgentWS upgrades to WebSocket and bridges to the agent's IO.
func (h *Handler) handleAgentWS(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("id")
	if agentID == "" {
		http.Error(w, "agent ID required", http.StatusBadRequest)
		return
	}

	bridge, ok := h.sessions.GetBridge(agentID)
	if !ok {
		http.Error(w, "agent not found or not interactive", http.StatusNotFound)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins for local dev.
	})
	if err != nil {
		h.logger.Printf("[ws] accept error: %v", err)
		return
	}

	session, isPrimary := h.sessions.AddSession(r.Context(), agentID, conn)
	defer h.sessions.RemoveSession(agentID, session)

	h.logger.Printf("[ws] client connected to agent %s (primary=%v)", agentID, isPrimary)

	// Replay buffered output.
	for _, line := range bridge.Buffer.Lines() {
		if err := conn.Write(session.ctx, websocket.MessageText, line); err != nil {
			return
		}
	}

	// Send current status.
	_ = conn.Write(session.ctx, websocket.MessageText, StatusMsg("running", nil))

	// Start read loop for primary client (writes to agent stdin).
	if isPrimary {
		go h.readLoop(session, bridge)
	}

	// Wait for agent exit or client disconnect / server shutdown.
	// Session context is derived from request context, so server shutdown
	// automatically cancels the session.
	select {
	case <-bridge.Done:
		_ = conn.Write(session.ctx, websocket.MessageText, StatusMsg("completed", intPtr(0)))
		conn.Close(websocket.StatusNormalClosure, "agent exited")
	case <-session.ctx.Done():
		// Client disconnected or server shutting down — agent continues running.
	}
}

// readLoop reads messages from the WebSocket client and writes to the agent's stdin.
func (h *Handler) readLoop(session *Session, bridge *Bridge) {
	for {
		_, data, err := session.conn.Read(session.ctx)
		if err != nil {
			return // client disconnected
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = session.conn.Write(session.ctx, websocket.MessageText,
				ErrorMsg("invalid message format"))
			continue
		}

		if msg.Type != MsgInput {
			continue
		}

		text := strings.TrimSpace(msg.Text)
		if text == "" {
			continue
		}

		if err := bridge.WriteInput(text); err != nil {
			_ = session.conn.Write(session.ctx, websocket.MessageText,
				ErrorMsg("failed to send input to agent"))
		}
	}
}

func intPtr(v int) *int { return &v }
