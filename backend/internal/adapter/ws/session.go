package ws

import (
	"context"
	"io"
	"strings"
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

// ConnectionCallback is called when an agent's WebSocket connection count changes.
type ConnectionCallback func(agentID string)

// SessionManager tracks active WebSocket sessions per agent.
// It supports multiple observers per agent (first is read-write, rest are read-only).
type SessionManager struct {
	mu           sync.RWMutex
	sessions     map[string][]*Session // agentID → list of sessions
	bridges      map[string]*Bridge    // agentID → bridge (agent IO)
	onDisconnect ConnectionCallback    // called when agent sessions drop to zero
	onReconnect  ConnectionCallback    // called when agent sessions go from zero to non-zero
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string][]*Session),
		bridges:  make(map[string]*Bridge),
	}
}

// SetOnDisconnect sets the callback fired when an agent's last session disconnects.
func (sm *SessionManager) SetOnDisconnect(fn ConnectionCallback) {
	sm.onDisconnect = fn
}

// SetOnReconnect sets the callback fired when an agent gains its first session.
func (sm *SessionManager) SetOnReconnect(fn ConnectionCallback) {
	sm.onReconnect = fn
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
// The parent context should be derived from the HTTP request so that
// server shutdown automatically cancels all sessions.
// Returns true if this is the primary (read-write) session.
// Fires OnReconnect when sessions go from zero to non-zero.
func (sm *SessionManager) AddSession(parent context.Context, agentID string, conn *websocket.Conn) (*Session, bool) {
	ctx, cancel := context.WithCancel(parent)
	s := &Session{
		agentID: agentID,
		conn:    conn,
		ctx:     ctx,
		cancel:  cancel,
	}
	sm.mu.Lock()
	isPrimary := len(sm.sessions[agentID]) == 0
	sm.sessions[agentID] = append(sm.sessions[agentID], s)
	sm.mu.Unlock()
	if isPrimary && sm.onReconnect != nil {
		sm.onReconnect(agentID)
	}
	return s, isPrimary
}

// RemoveSession removes a WebSocket session.
// Fires OnDisconnect when the last session for an agent is removed.
func (sm *SessionManager) RemoveSession(agentID string, s *Session) {
	s.cancel()
	var fireDisconnect bool
	sm.mu.Lock()
	sessions := sm.sessions[agentID]
	for i, ss := range sessions {
		if ss == s {
			sm.sessions[agentID] = append(sessions[:i], sessions[i+1:]...)
			break
		}
	}
	if len(sm.sessions[agentID]) == 0 {
		delete(sm.sessions, agentID)
		fireDisconnect = true
	}
	sm.mu.Unlock()
	if fireDisconnect && sm.onDisconnect != nil {
		sm.onDisconnect(agentID)
	}
}

// SessionCount returns the number of active WebSocket sessions for an agent.
func (sm *SessionManager) SessionCount(agentID string) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions[agentID])
}

// BroadcastToAgent sends a message to all WebSocket clients observing an agent.
// Stale sessions (cancelled context) are removed during broadcast.
func (sm *SessionManager) BroadcastToAgent(agentID string, msg []byte) {
	sm.mu.RLock()
	sessions := make([]*Session, len(sm.sessions[agentID]))
	copy(sessions, sm.sessions[agentID])
	sm.mu.RUnlock()

	var stale []*Session
	for _, s := range sessions {
		if s.ctx.Err() != nil {
			stale = append(stale, s)
			continue
		}
		_ = s.conn.Write(s.ctx, websocket.MessageText, msg)
	}

	if len(stale) > 0 {
		sm.mu.Lock()
		for _, s := range stale {
			list := sm.sessions[agentID]
			for i, ss := range list {
				if ss == s {
					sm.sessions[agentID] = append(list[:i], list[i+1:]...)
					break
				}
			}
		}
		if len(sm.sessions[agentID]) == 0 {
			delete(sm.sessions, agentID)
		}
		sm.mu.Unlock()
	}
}

// CloseAllSessions cancels every active session context and clears the session map.
// Used during graceful server shutdown to ensure all WebSocket connections are closed.
func (sm *SessionManager) CloseAllSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for agentID, sessions := range sm.sessions {
		for _, s := range sessions {
			s.cancel()
		}
		delete(sm.sessions, agentID)
	}
}

// InputHandler processes user input for an agent. SDK-based agents use this
// to route input through client.Query() instead of raw stdin pipes.
type InputHandler func(text string) error

// Bridge manages the IO between an interactive agent process and WebSocket clients.
type Bridge struct {
	AgentID          string
	Stdin            io.WriteCloser  // write to agent's stdin (legacy)
	InputHandler     InputHandler    // SDK-based input handler (takes precedence over Stdin)
	InterruptHandler func()          // called to interrupt the current agent turn
	Buffer           *RingBuffer     // output ring buffer for reconnection
	Done             <-chan struct{} // closed when agent exits
	mu               sync.Mutex
	inputQueue       []string // queued input messages sent during an active turn
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

// Interrupt interrupts the current agent turn. No-op if no handler is set.
func (b *Bridge) Interrupt() {
	if b.InterruptHandler != nil {
		b.InterruptHandler()
	}
}

// WriteInput sends user input to the agent.
// Uses InputHandler if set (SDK mode), otherwise writes to Stdin (legacy mode).
// If the InputHandler returns "turn already in progress", the input is queued
// and will be sent when DrainQueue is called after the turn completes.
func (b *Bridge) WriteInput(text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.InputHandler != nil {
		err := b.InputHandler(text)
		if err != nil && strings.Contains(err.Error(), "turn already in progress") {
			b.inputQueue = append(b.inputQueue, text)
			return nil
		}
		return err
	}
	if b.Stdin != nil {
		_, err := io.WriteString(b.Stdin, text+"\n")
		return err
	}
	return nil
}

// DrainQueue sends the first queued input message (if any) via the InputHandler.
// It should be called after a turn completes to deliver input that arrived
// during the previous turn. Returns true if a queued message was sent.
func (b *Bridge) DrainQueue() bool {
	b.mu.Lock()
	if len(b.inputQueue) == 0 || b.InputHandler == nil {
		b.mu.Unlock()
		return false
	}
	text := b.inputQueue[0]
	b.inputQueue = b.inputQueue[1:]
	b.mu.Unlock()
	// Send outside lock — InputHandler may block.
	_ = b.InputHandler(text)
	return true
}

// QueueDepth returns the number of queued input messages.
func (b *Bridge) QueueDepth() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.inputQueue)
}

// StartOutputRelay reads from an output channel (from InteractiveAgent) and
// broadcasts each message to all WebSocket clients while buffering for reconnection.
// The relay stops when ctx is cancelled or the output channel is closed.
func (sm *SessionManager) StartOutputRelay(ctx context.Context, agentID string, output <-chan []byte) {
	bridge, ok := sm.GetBridge(agentID)
	if !ok {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case text, ok := <-output:
			if !ok {
				return
			}
			msg := OutputMsg(string(text))
			bridge.Buffer.Write(msg)
			sm.BroadcastToAgent(agentID, msg)
		}
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
