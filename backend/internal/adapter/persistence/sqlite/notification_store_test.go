package sqlite

import (
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

func TestNotificationStore_InsertAndListActive(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewNotificationStore(db)

	now := time.Now().UTC().Truncate(time.Millisecond)
	n1 := domain.Notification{ID: "n-1", AgentID: "agent-1", Title: "Agent 1 needs attention", Body: "waiting for input", CreatedAt: now.Add(-2 * time.Minute)}
	n2 := domain.Notification{ID: "n-2", AgentID: "agent-2", Title: "Agent 2 needs attention", Body: "waiting for input", CreatedAt: now.Add(-1 * time.Minute)}
	acked := now
	n3 := domain.Notification{ID: "n-3", AgentID: "agent-3", Title: "Agent 3 needs attention", Body: "done", CreatedAt: now, AcknowledgedAt: &acked}

	for _, n := range []domain.Notification{n1, n2, n3} {
		if err := store.Insert(n); err != nil {
			t.Fatalf("Insert %s: %v", n.ID, err)
		}
	}

	// ListActive should return only unacknowledged, newest first.
	items, err := store.ListActive("")
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("want 2 active, got %d", len(items))
	}
	if items[0].ID != "n-2" {
		t.Errorf("first item: want n-2, got %s", items[0].ID)
	}
	if items[1].ID != "n-1" {
		t.Errorf("second item: want n-1, got %s", items[1].ID)
	}
}

func TestNotificationStore_ListActiveByAgent(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewNotificationStore(db)

	now := time.Now().UTC()
	store.Insert(domain.Notification{ID: "n-1", AgentID: "agent-1", Title: "t", Body: "b", CreatedAt: now})
	store.Insert(domain.Notification{ID: "n-2", AgentID: "agent-2", Title: "t", Body: "b", CreatedAt: now})

	items, err := store.ListActive("agent-1")
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("want 1, got %d", len(items))
	}
	if items[0].AgentID != "agent-1" {
		t.Errorf("agent_id: want agent-1, got %s", items[0].AgentID)
	}
}

func TestNotificationStore_Acknowledge(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewNotificationStore(db)

	now := time.Now().UTC()
	store.Insert(domain.Notification{ID: "n-1", AgentID: "agent-1", Title: "t", Body: "b", CreatedAt: now})

	if err := store.Acknowledge("n-1"); err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}

	items, _ := store.ListActive("")
	if len(items) != 0 {
		t.Errorf("want 0 active after ack, got %d", len(items))
	}

	// Acknowledging non-existent returns error.
	if err := store.Acknowledge("does-not-exist"); err == nil {
		t.Error("expected error for non-existent notification")
	}
}

func TestNotificationStore_DeleteForAgent(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewNotificationStore(db)

	now := time.Now().UTC()
	store.Insert(domain.Notification{ID: "n-1", AgentID: "agent-1", Title: "t", Body: "b", CreatedAt: now})
	store.Insert(domain.Notification{ID: "n-2", AgentID: "agent-1", Title: "t", Body: "b", CreatedAt: now.Add(time.Minute)})
	store.Insert(domain.Notification{ID: "n-3", AgentID: "agent-2", Title: "t", Body: "b", CreatedAt: now})

	if err := store.DeleteForAgent("agent-1"); err != nil {
		t.Fatalf("DeleteForAgent: %v", err)
	}

	items, _ := store.ListActive("")
	if len(items) != 1 {
		t.Fatalf("want 1, got %d", len(items))
	}
	if items[0].AgentID != "agent-2" {
		t.Errorf("remaining should be agent-2, got %s", items[0].AgentID)
	}
}

func TestNotificationStore_FindActiveByAgent(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	store := NewNotificationStore(db)

	now := time.Now().UTC()
	store.Insert(domain.Notification{ID: "n-1", AgentID: "agent-1", Title: "t", Body: "b", CreatedAt: now})

	n, err := store.FindActiveByAgent("agent-1")
	if err != nil {
		t.Fatalf("FindActiveByAgent: %v", err)
	}
	if n == nil || n.ID != "n-1" {
		t.Errorf("want n-1, got %v", n)
	}

	// No active notification for unknown agent.
	n, err = store.FindActiveByAgent("agent-999")
	if err != nil {
		t.Fatalf("FindActiveByAgent unknown: %v", err)
	}
	if n != nil {
		t.Errorf("want nil, got %+v", n)
	}
}
