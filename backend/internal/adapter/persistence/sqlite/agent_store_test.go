package sqlite

import (
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

func TestAgentStore_AddAndFind(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewAgentStore(db)

	info := domain.AgentInfo{
		ID:        "agent-abc123",
		Role:      "developer",
		Ref:       "my-track",
		Status:    "running",
		PID:       12345,
		StartedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
	}
	store.AddAgent(info)

	got, err := store.FindAgent("agent-abc123")
	if err != nil {
		t.Fatalf("FindAgent: %v", err)
	}
	if got.ID != info.ID {
		t.Errorf("ID: want %q, got %q", info.ID, got.ID)
	}
	if got.Role != "developer" {
		t.Errorf("Role: want developer, got %q", got.Role)
	}
}

func TestAgentStore_FindByPrefix(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewAgentStore(db)

	store.AddAgent(domain.AgentInfo{
		ID: "agent-abc123", Role: "developer", Ref: "t1", Status: "running",
		StartedAt: time.Now(), UpdatedAt: time.Now(),
	})

	got, err := store.FindAgent("agent-abc")
	if err != nil {
		t.Fatalf("FindAgent prefix: %v", err)
	}
	if got.ID != "agent-abc123" {
		t.Errorf("ID: want agent-abc123, got %q", got.ID)
	}
}

func TestAgentStore_UpdateStatus(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewAgentStore(db)

	store.AddAgent(domain.AgentInfo{
		ID: "agent-upd", Role: "developer", Ref: "t1", Status: "running",
		StartedAt: time.Now(), UpdatedAt: time.Now(),
	})

	store.UpdateStatus("agent-upd", "halted")

	got, err := store.FindAgent("agent-upd")
	if err != nil {
		t.Fatalf("FindAgent: %v", err)
	}
	if got.Status != "halted" {
		t.Errorf("Status: want halted, got %q", got.Status)
	}
}

func TestAgentStore_AgentsByStatus(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewAgentStore(db)

	store.AddAgent(domain.AgentInfo{ID: "a1", Role: "developer", Ref: "t1", Status: "running", StartedAt: time.Now(), UpdatedAt: time.Now()})
	store.AddAgent(domain.AgentInfo{ID: "a2", Role: "developer", Ref: "t2", Status: "halted", StartedAt: time.Now(), UpdatedAt: time.Now()})
	store.AddAgent(domain.AgentInfo{ID: "a3", Role: "reviewer", Ref: "t3", Status: "running", StartedAt: time.Now(), UpdatedAt: time.Now()})

	running := store.AgentsByStatus("running")
	if len(running) != 2 {
		t.Errorf("AgentsByStatus(running): want 2, got %d", len(running))
	}
}

func TestAgentStore_FindByRef(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewAgentStore(db)

	store.AddAgent(domain.AgentInfo{ID: "old", Role: "developer", Ref: "track-1", Status: "halted", StartedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Now()})
	store.AddAgent(domain.AgentInfo{ID: "new", Role: "developer", Ref: "track-1", Status: "running", StartedAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), UpdatedAt: time.Now()})

	got := store.FindByRef("track-1")
	if got == nil {
		t.Fatal("FindByRef: nil")
	}
	if got.ID != "new" {
		t.Errorf("ID: want new, got %q", got.ID)
	}
}

func TestAgentStore_Agents(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewAgentStore(db)

	store.AddAgent(domain.AgentInfo{ID: "a1", Role: "developer", Ref: "t1", Status: "running", StartedAt: time.Now(), UpdatedAt: time.Now()})
	store.AddAgent(domain.AgentInfo{ID: "a2", Role: "reviewer", Ref: "t2", Status: "halted", StartedAt: time.Now(), UpdatedAt: time.Now()})

	all := store.Agents()
	if len(all) != 2 {
		t.Errorf("Agents: want 2, got %d", len(all))
	}
}
