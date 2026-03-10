package agent

import (
	"sync"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

// mockSuspender records SuspendAgent calls.
type mockSuspender struct {
	mu        sync.Mutex
	suspended []string
}

func (m *mockSuspender) SuspendAgent(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suspended = append(m.suspended, id)
	return nil
}

func (m *mockSuspender) getSuspended() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.suspended))
	copy(out, m.suspended)
	return out
}

// mockRoleLookup returns a fixed role for a given agent.
type mockRoleLookup struct {
	agents map[string]string // agentID → role
}

func (m *mockRoleLookup) FindAgent(id string) (*domain.AgentInfo, error) {
	role, ok := m.agents[id]
	if !ok {
		return nil, domain.ErrAgentNotFound
	}
	return &domain.AgentInfo{ID: id, Role: role}, nil
}

func TestConnectionSuspender_GracePeriodExpiry(t *testing.T) {
	t.Parallel()

	suspender := &mockSuspender{}
	lookup := &mockRoleLookup{agents: map[string]string{"agent-1": "interactive"}}
	cs := NewConnectionSuspender(suspender, lookup, 50*time.Millisecond)
	defer cs.Stop()

	cs.OnAgentDisconnected("agent-1")

	// Wait for grace period to expire.
	time.Sleep(100 * time.Millisecond)

	if got := suspender.getSuspended(); len(got) != 1 || got[0] != "agent-1" {
		t.Errorf("expected [agent-1] suspended, got %v", got)
	}
}

func TestConnectionSuspender_ReconnectCancelsTimer(t *testing.T) {
	t.Parallel()

	suspender := &mockSuspender{}
	lookup := &mockRoleLookup{agents: map[string]string{"agent-2": "interactive"}}
	cs := NewConnectionSuspender(suspender, lookup, 100*time.Millisecond)
	defer cs.Stop()

	cs.OnAgentDisconnected("agent-2")

	// Reconnect before grace period.
	time.Sleep(30 * time.Millisecond)
	cs.OnAgentReconnected("agent-2")

	// Wait past grace period.
	time.Sleep(150 * time.Millisecond)

	if got := suspender.getSuspended(); len(got) != 0 {
		t.Errorf("expected no suspensions after reconnect, got %v", got)
	}
}

func TestConnectionSuspender_WorkerRolesIgnored(t *testing.T) {
	t.Parallel()

	suspender := &mockSuspender{}
	lookup := &mockRoleLookup{agents: map[string]string{
		"dev-1": "developer",
		"rev-1": "reviewer",
	}}
	cs := NewConnectionSuspender(suspender, lookup, 50*time.Millisecond)
	defer cs.Stop()

	cs.OnAgentDisconnected("dev-1")
	cs.OnAgentDisconnected("rev-1")

	time.Sleep(100 * time.Millisecond)

	if got := suspender.getSuspended(); len(got) != 0 {
		t.Errorf("expected no suspensions for worker roles, got %v", got)
	}
}

func TestConnectionSuspender_ZeroGracePeriodDisables(t *testing.T) {
	t.Parallel()

	suspender := &mockSuspender{}
	lookup := &mockRoleLookup{agents: map[string]string{"agent-3": "interactive"}}
	cs := NewConnectionSuspender(suspender, lookup, 0)
	defer cs.Stop()

	cs.OnAgentDisconnected("agent-3")

	time.Sleep(50 * time.Millisecond)

	if got := suspender.getSuspended(); len(got) != 0 {
		t.Errorf("expected no suspensions when disabled, got %v", got)
	}
}

func TestConnectionSuspender_AdvisorRolesSuspended(t *testing.T) {
	t.Parallel()

	suspender := &mockSuspender{}
	lookup := &mockRoleLookup{agents: map[string]string{"adv-1": "advisor-product"}}
	cs := NewConnectionSuspender(suspender, lookup, 50*time.Millisecond)
	defer cs.Stop()

	cs.OnAgentDisconnected("adv-1")

	time.Sleep(100 * time.Millisecond)

	if got := suspender.getSuspended(); len(got) != 1 || got[0] != "adv-1" {
		t.Errorf("expected [adv-1] suspended, got %v", got)
	}
}
