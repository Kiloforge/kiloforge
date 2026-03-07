package agent

import (
	"testing"
	"time"

	"crelay/internal/core/domain"
	"crelay/internal/core/testutil"
)

func TestShutdownAll_NoRunningAgents(t *testing.T) {
	store := &testutil.MockAgentStore{}
	sm := NewShutdownManager(store)
	result := sm.ShutdownAll(1 * time.Second)

	if len(result.Suspended) != 0 || len(result.ForceKilled) != 0 {
		t.Errorf("expected empty result, got suspended=%d force_killed=%d",
			len(result.Suspended), len(result.ForceKilled))
	}
}

func TestShutdownAll_NoPID(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "agent-1", Status: "running", PID: 0},
		},
	}
	sm := NewShutdownManager(store)
	result := sm.ShutdownAll(1 * time.Second)

	if len(result.NoPID) != 1 || result.NoPID[0] != "agent-1" {
		t.Errorf("expected agent-1 in NoPID, got %v", result.NoPID)
	}
	agent, _ := store.FindAgent("agent-1")
	if agent.Status != "suspended" {
		t.Errorf("expected suspended, got %s", agent.Status)
	}
}

func TestShutdownAll_DeadProcess(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "agent-1", Status: "running", PID: 999999},
		},
	}
	sm := NewShutdownManager(store)
	result := sm.ShutdownAll(1 * time.Second)

	if len(result.AlreadyDead) != 1 {
		t.Errorf("expected 1 already dead, got %d", len(result.AlreadyDead))
	}
	agent, _ := store.FindAgent("agent-1")
	if agent.Status != "suspended" {
		t.Errorf("expected suspended, got %s", agent.Status)
	}
}

func TestShutdownAll_MixedStates(t *testing.T) {
	store := &testutil.MockAgentStore{
		AgentData: []domain.AgentInfo{
			{ID: "agent-nopid", Status: "running", PID: 0},
			{ID: "agent-dead", Status: "waiting", PID: 999999},
			{ID: "agent-completed", Status: "completed", PID: 123},
		},
	}
	sm := NewShutdownManager(store)
	result := sm.ShutdownAll(1 * time.Second)

	if len(result.NoPID) != 1 {
		t.Errorf("expected 1 NoPID, got %d", len(result.NoPID))
	}
	if len(result.AlreadyDead) != 1 {
		t.Errorf("expected 1 AlreadyDead, got %d", len(result.AlreadyDead))
	}
	// completed agent should not be touched
	completed, _ := store.FindAgent("agent-completed")
	if completed.Status != "completed" {
		t.Errorf("expected completed to stay completed, got %s", completed.Status)
	}
}
