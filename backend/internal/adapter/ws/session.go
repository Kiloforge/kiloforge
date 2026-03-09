package ws

import (
	"context"
	"io"
	"sync"

	"nhooyr.io/websocket"
)

// Session represents an active WebSocket connection to an interactive agent.
type Session struct {
	agentID string
	conn    *websocket.Conn
	ctx     context.Context
	cancel  context.CancelFunc
}

// SessionManager tracks active WebSocket sessions per agent.
// It supports multiple observers per agent (first is read-write, rest are read-only).
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string][]*Session // agentID → list of sessions
	bridges  map[string]*Bridge    // agentID → bridge (agent IO)
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string][]*Session),
		bridges:  make(map[string]*Bridge),
	}
}

// RegisterBridge registers an agent's IO bridge so WebSocket sessions can connect.
func (sm *SessionManager) RegisterBridge(agentID string, bridge *Bridge) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.bridges[agentID] = bridge
}

// UnregisterBridge removes an agent's bridge.
func (sm *SessionManager) UnregisterBridge(agentID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.bridges, agentID)
}

// GetBridge returns the bridge for an agent.
func (sm *SessionManager) GetBridge(agentID string) (*Bridge, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	b, ok := sm.bridges[agentID]
	return b, ok
}

// AddSession registers a WebSocket connection for an agent.
// Returns true if this is the primary (read-write) session.
func (sm *SessionManager) AddSession(agentID string, conn *websocket.Conn) (*Session, bool) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{
		agentID: agentID,
		conn:    conn,
		ctx:     ctx,
		cancel:  cancel,
	}
	sm.mu.Lock()
	defer sm.mu.Unlock()
	isPrimary := len(sm.sessions[agentID]) == 0
	sm.sessions[agentID] = append(sm.sessions[agentID], s)
	return s, isPrimary
}

// RemoveSession removes a WebSocket session.
func (sm *SessionManager) RemoveSession(agentID string, s *Session) {
	s.cancel()
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sessions := sm.sessions[agentID]
	for i, ss := range sessions {
		if ss == s {
			sm.sessions[agentID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}
	if len(sm.sessions[agentID]) == 0 {
		delete(sm.sessions, agentID)
	}
}

// BroadcastToAgent sends a message to all WebSocket clients observing an agent.
func (sm *SessionManager) BroadcastToAgent(agentID string, msg []byte) {
	sm.mu.RLock()
	sessions := sm.sessions[agentID]
	sm.mu.RUnlock()

	for _, s := range sessions {
		_ = s.conn.Write(s.ctx, websocket.MessageText, msg)
	}
}

// InputHandler processes user input for an agent. SDK-based agents use this
// to route input through client.Query() instead of raw stdin pipes.
type InputHandler func(text string) error

// Bridge manages the IO between an interactive agent process and WebSocket clients.
type Bridge struct {
	AgentID      string
	Stdin        io.WriteCloser // write to agent's stdin (legacy)
	InputHandler InputHandler   // SDK-based input handler (takes precedence over Stdin)
	Buffer       *RingBuffer    // output ring buffer for reconnection
	Done         <-chan struct{} // closed when agent exits
	mu           sync.Mutex
}

// NewBridge creates a new bridge for an interactive agent.
func NewBridge(agentID string, stdin io.WriteCloser, done <-chan struct{}) *Bridge {
	return &Bridge{
		AgentID: agentID,
		Stdin:   stdin,
		Buffer:  NewRingBuffer(500),
		Done:    done,
	}
}

// NewSDKBridge creates a bridge for an SDK-based interactive agent.
// Input is routed through the InputHandler instead of a raw stdin pipe.
func NewSDKBridge(agentID string, handler InputHandler, done <-chan struct{}) *Bridge {
	return &Bridge{
		AgentID:      agentID,
		InputHandler: handler,
		Buffer:       NewRingBuffer(500),
		Done:         done,
	}
}

// WriteInput sends user input to the agent.
// Uses InputHandler if set (SDK mode), otherwise writes to Stdin (legacy mode).
func (b *Bridge) WriteInput(text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.InputHandler != nil {
		return b.InputHandler(text)
	}
	if b.Stdin != nil {
		_, err := io.WriteString(b.Stdin, text+"\n")
		return err
	}
	return nil
}

// StartOutputRelay reads from an output channel (from InteractiveAgent) and
// broadcasts each message to all WebSocket clients while buffering for reconnection.
// It runs until the output channel is closed.
func (sm *SessionManager) StartOutputRelay(agentID string, output <-chan []byte) {
	bridge, ok := sm.GetBridge(agentID)
	if !ok {
		return
	}
	for text := range output {
		msg := OutputMsg(string(text))
		bridge.Buffer.Write(msg)
		sm.BroadcastToAgent(agentID, msg)
	}
}

// StartStructuredRelay reads pre-serialized JSON messages from a channel
// and broadcasts them to WebSocket clients. Used by SDK-based agents that
// produce structured messages (turn_start, text, tool_use, etc.).
// The relay stops when ctx is cancelled or the messages channel is closed.
func (sm *SessionManager) StartStructuredRelay(ctx context.Context, agentID string, messages <-chan []byte) {
	bridge, ok := sm.GetBridge(agentID)
	if !ok {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			bridge.Buffer.Write(msg)
			sm.BroadcastToAgent(agentID, msg)
		}
	}
}
