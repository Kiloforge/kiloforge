package service

import (
	"testing"

	"kiloforge/internal/core/domain"
)

// stubNotificationStore is an in-memory store for testing.
type stubNotificationStore struct {
	notifications []domain.Notification
}

func (s *stubNotificationStore) Insert(n domain.Notification) error {
	s.notifications = append(s.notifications, n)
	return nil
}

func (s *stubNotificationStore) ListActive(agentID string) ([]domain.Notification, error) {
	var result []domain.Notification
	for _, n := range s.notifications {
		if n.AcknowledgedAt != nil {
			continue
		}
		if agentID != "" && n.AgentID != agentID {
			continue
		}
		result = append(result, n)
	}
	return result, nil
}

func (s *stubNotificationStore) Acknowledge(id string) error {
	for i := range s.notifications {
		if s.notifications[i].ID == id {
			now := s.notifications[i].CreatedAt
			s.notifications[i].AcknowledgedAt = &now
			return nil
		}
	}
	return nil
}

func (s *stubNotificationStore) DeleteForAgent(agentID string) error {
	var kept []domain.Notification
	for _, n := range s.notifications {
		if n.AgentID != agentID {
			kept = append(kept, n)
		}
	}
	s.notifications = kept
	return nil
}

func (s *stubNotificationStore) FindActiveByAgent(agentID string) (*domain.Notification, error) {
	for _, n := range s.notifications {
		if n.AgentID == agentID && n.AcknowledgedAt == nil {
			return &n, nil
		}
	}
	return nil, nil
}

// stubEventBus captures published events.
type stubEventBus struct {
	events []domain.Event
}

func (b *stubEventBus) Publish(e domain.Event)                { b.events = append(b.events, e) }
func (b *stubEventBus) Subscribe() <-chan domain.Event         { return nil }
func (b *stubEventBus) Unsubscribe(_ <-chan domain.Event)      {}
func (b *stubEventBus) ClientCount() int                       { return 0 }

func TestNotificationService_Create_Deduplicates(t *testing.T) {
	store := &stubNotificationStore{}
	bus := &stubEventBus{}
	svc := NewNotificationService(store, bus)

	// First create should succeed.
	err := svc.Create("agent-1", "Agent 1 needs your attention", "waiting")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(store.notifications) != 1 {
		t.Fatalf("want 1 notification, got %d", len(store.notifications))
	}
	if len(bus.events) != 1 || bus.events[0].Type != domain.EventNotificationCreated {
		t.Errorf("want notification_created event, got %v", bus.events)
	}

	// Second create for same agent should be a no-op (deduplicate).
	bus.events = nil
	err = svc.Create("agent-1", "Agent 1 needs your attention", "waiting")
	if err != nil {
		t.Fatalf("Create duplicate: %v", err)
	}
	if len(store.notifications) != 1 {
		t.Errorf("want still 1 notification, got %d", len(store.notifications))
	}
	if len(bus.events) != 0 {
		t.Errorf("want no events for duplicate, got %d", len(bus.events))
	}
}

func TestNotificationService_DismissForAgent(t *testing.T) {
	store := &stubNotificationStore{}
	bus := &stubEventBus{}
	svc := NewNotificationService(store, bus)

	svc.Create("agent-1", "t", "b")
	bus.events = nil

	err := svc.DismissForAgent("agent-1")
	if err != nil {
		t.Fatalf("DismissForAgent: %v", err)
	}

	items, _ := store.ListActive("")
	if len(items) != 0 {
		t.Errorf("want 0 active, got %d", len(items))
	}
	if len(bus.events) != 1 || bus.events[0].Type != domain.EventNotificationDismissed {
		t.Errorf("want notification_dismissed event, got %v", bus.events)
	}
}

func TestNotificationService_DismissForAgent_NopWhenNone(t *testing.T) {
	store := &stubNotificationStore{}
	bus := &stubEventBus{}
	svc := NewNotificationService(store, bus)

	// Dismissing when no active notification should be a no-op.
	err := svc.DismissForAgent("agent-1")
	if err != nil {
		t.Fatalf("DismissForAgent: %v", err)
	}
	if len(bus.events) != 0 {
		t.Errorf("want no events, got %d", len(bus.events))
	}
}

func TestNotificationService_Acknowledge(t *testing.T) {
	store := &stubNotificationStore{}
	bus := &stubEventBus{}
	svc := NewNotificationService(store, bus)

	svc.Create("agent-1", "t", "b")
	id := store.notifications[0].ID
	bus.events = nil

	err := svc.Acknowledge(id)
	if err != nil {
		t.Fatalf("Acknowledge: %v", err)
	}

	items, _ := store.ListActive("")
	if len(items) != 0 {
		t.Errorf("want 0 active, got %d", len(items))
	}
}

func TestNotificationService_CleanTerminalAgents(t *testing.T) {
	store := &stubNotificationStore{}
	bus := &stubEventBus{}
	svc := NewNotificationService(store, bus)

	svc.Create("agent-1", "t", "b")
	svc.Create("agent-2", "t", "b")
	bus.events = nil

	err := svc.CleanForAgent("agent-1")
	if err != nil {
		t.Fatalf("CleanForAgent: %v", err)
	}

	items, _ := store.ListActive("")
	if len(items) != 1 {
		t.Fatalf("want 1 active, got %d", len(items))
	}
	if items[0].AgentID != "agent-2" {
		t.Errorf("remaining should be agent-2, got %s", items[0].AgentID)
	}
	// Should have published dismissed event.
	if len(bus.events) != 1 || bus.events[0].Type != domain.EventNotificationDismissed {
		t.Errorf("want notification_dismissed event, got %v", bus.events)
	}
}
