package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"kiloforge/internal/core/domain"

	"nhooyr.io/websocket"
)

// AgentFinder looks up agent info by ID prefix. The WS handler uses this
// to resolve agent status when no bridge is registered (agent already exited).
type AgentFinder interface {
	FindAgent(idPrefix string) (*domain.AgentInfo, error)
}

// Handler handles WebSocket upgrade requests for interactive agent sessions.
type Handler struct {
	sessions *SessionManager
	agents   AgentFinder
	logger   *log.Logger
}

// NewHandler creates a new WebSocket handler.
// agents may be nil — if so, the handler cannot resolve status for exited agents.
func NewHandler(sessions *SessionManager, agents AgentFinder, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{sessions: sessions, agents: agents, logger: logger}
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
		// No bridge — agent may have exited. If we can look it up, send its
		// terminal status over WebSocket so the client knows to stop retrying.
		h.handleNoBridge(w, r, agentID)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins for local dev.
	})
	if err != nil {
		h.logger.Printf("[ws] accept error for agent %s: %v", agentID, err)
		return
	}

	session, isPrimary := h.sessions.AddSession(r.Context(), agentID, conn)
	defer h.sessions.RemoveSession(agentID, session)

	h.logger.Printf("[ws] client connected to agent %s (primary=%v)", agentID, isPrimary)

	// Replay buffered output.
	for _, line := range bridge.Buffer.Lines() {
		if err := conn.Write(session.ctx, websocket.MessageText, line); err != nil {
			h.logger.Printf("[ws] replay write error for agent %s: %v", agentID, err)
			return
		}
	}

	// Send actual agent status instead of hardcoding "running".
	initialStatus := "running"
	if h.agents != nil {
		if info, err := h.agents.FindAgent(agentID); err == nil && info != nil {
			initialStatus = info.Status
		}
	}
	_ = conn.Write(session.ctx, websocket.MessageText, StatusMsg(initialStatus, nil))

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
		_ = conn.Close(websocket.StatusNormalClosure, "agent exited")
		h.logger.Printf("[ws] agent %s exited, closing WebSocket", agentID)
	case <-session.ctx.Done():
		h.logger.Printf("[ws] client disconnected from agent %s", agentID)
	}
}

// handleNoBridge handles WebSocket connections when no bridge is registered.
// If the agent exists in the store and is in a terminal state, it upgrades the
// connection, sends the terminal status, and closes cleanly. This prevents
// the client from entering an infinite reconnect loop.
func (h *Handler) handleNoBridge(w http.ResponseWriter, r *http.Request, agentID string) {
	if h.agents == nil {
		http.Error(w, "agent not found or not interactive", http.StatusNotFound)
		return
	}

	info, err := h.agents.FindAgent(agentID)
	if err != nil || info == nil {
		http.Error(w, "agent not found or not interactive", http.StatusNotFound)
		return
	}

	if !info.IsTerminal() {
		// Agent exists but isn't terminal and has no bridge — transient state.
		http.Error(w, "agent not ready", http.StatusServiceUnavailable)
		return
	}

	// Agent is in a terminal state. Upgrade the WebSocket, send the status,
	// and close cleanly so the client stops reconnecting.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Printf("[ws] accept error for terminal agent %s: %v", agentID, err)
		return
	}

	h.logger.Printf("[ws] client connected to terminal agent %s (status=%s)", agentID, info.Status)
	ctx := r.Context()
	_ = conn.Write(ctx, websocket.MessageText, StatusMsg(info.Status, nil))
	_ = conn.Close(websocket.StatusNormalClosure, "agent already "+info.Status)
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

		switch msg.Type {
		case MsgInterrupt:
			bridge.Interrupt()

		case MsgInput:
			text := strings.TrimSpace(msg.Text)
			if text == "" {
				continue
			}
			if err := bridge.WriteInput(text); err != nil {
				_ = session.conn.Write(session.ctx, websocket.MessageText,
					ErrorMsg(fmt.Sprintf("failed to send input to agent: %v", err)))
			}

		default:
			continue
		}
	}
}

func intPtr(v int) *int { return &v }
