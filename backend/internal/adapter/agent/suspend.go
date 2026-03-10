package agent

import (
	"fmt"
	"os"
	"sync"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// AgentSuspender suspends an interactive agent by ID.
type AgentSuspender interface {
	SuspendAgent(id string) error
}

// AgentRoleLookup returns the role for an agent by ID.
type AgentRoleLookup interface {
	FindAgent(id string) (*domain.AgentInfo, error)
}

// ConnectionSuspender coordinates grace-period timers for idle agent suspension.
// When all browser connections drop, a timer starts. If no reconnection occurs
// before the grace period expires, the agent is suspended.
type ConnectionSuspender struct {
	suspender    AgentSuspender
	roleLookup   AgentRoleLookup
	gracePeriod  time.Duration
	eventBus     port.EventBus

	mu     sync.Mutex
	timers map[string]*time.Timer // agentID → pending suspension timer
}

// NewConnectionSuspender creates a new idle-disconnect coordinator.
// A gracePeriod of 0 disables auto-suspension.
func NewConnectionSuspender(suspender AgentSuspender, roleLookup AgentRoleLookup, gracePeriod time.Duration) *ConnectionSuspender {
	return &ConnectionSuspender{
		suspender:   suspender,
		roleLookup:  roleLookup,
		gracePeriod: gracePeriod,
		timers:      make(map[string]*time.Timer),
	}
}

// SetEventBus sets the event bus for publishing events.
func (cs *ConnectionSuspender) SetEventBus(eb port.EventBus) {
	cs.eventBus = eb
}

// OnAgentDisconnected starts the grace-period timer for an agent.
// Called when the last WebSocket session for an agent disconnects.
func (cs *ConnectionSuspender) OnAgentDisconnected(agentID string) {
	if cs.gracePeriod <= 0 {
		return
	}

	// Check if agent is a worker role — never auto-suspend workers.
	agent, err := cs.roleLookup.FindAgent(agentID)
	if err != nil {
		return
	}
	if domain.IsWorkerRole(agent.Role) {
		return
	}

	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Cancel any existing timer (shouldn't happen, but be safe).
	if t, ok := cs.timers[agentID]; ok {
		t.Stop()
	}

	cs.timers[agentID] = time.AfterFunc(cs.gracePeriod, func() {
		cs.mu.Lock()
		delete(cs.timers, agentID)
		cs.mu.Unlock()

		if err := cs.suspender.SuspendAgent(agentID); err != nil {
			fmt.Fprintf(os.Stderr, "[idle-suspend] failed to suspend %s: %v\n", agentID, err)
		}
	})
}

// OnAgentReconnected cancels the grace-period timer for an agent.
// Called when a new WebSocket session connects to an agent that had zero sessions.
func (cs *ConnectionSuspender) OnAgentReconnected(agentID string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if t, ok := cs.timers[agentID]; ok {
		t.Stop()
		delete(cs.timers, agentID)
	}
}

// Stop cancels all pending timers. Used during shutdown.
func (cs *ConnectionSuspender) Stop() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for id, t := range cs.timers {
		t.Stop()
		delete(cs.timers, id)
	}
}
