package agent

import (
	"context"
	"errors"
	"testing"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/core/domain"
)

func TestActiveCount_Empty(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 3}, nil, nil)
	if got := s.ActiveCount(); got != 0 {
		t.Errorf("ActiveCount() = %d, want 0", got)
	}
}

func TestActiveCount_WithAgents(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 3}, nil, nil)

	// Manually add agents to the map (simulates spawn).
	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeAgents["a2"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a2"}}
	s.activeMu.Unlock()

	if got := s.ActiveCount(); got != 2 {
		t.Errorf("ActiveCount() = %d, want 2", got)
	}
}

func TestCanSpawn_UnderLimit(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 3}, nil, nil)

	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeMu.Unlock()

	if !s.CanSpawn() {
		t.Error("CanSpawn() = false, want true (1/3 slots used)")
	}
}

func TestCanSpawn_AtLimit(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 2}, nil, nil)

	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeAgents["a2"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a2"}}
	s.activeMu.Unlock()

	if s.CanSpawn() {
		t.Error("CanSpawn() = true, want false (2/2 slots used)")
	}
}

func TestCanSpawn_DefaultConfig(t *testing.T) {
	t.Parallel()
	// MaxSwarmSize=0 should fall back to default of 3.
	s := NewSpawner(&config.Config{}, nil, nil)

	if !s.CanSpawn() {
		t.Error("CanSpawn() = false, want true (0/3 default)")
	}
}

func TestCanSpawn_LegacyMaxWorkers(t *testing.T) {
	t.Parallel()
	// Old config with only MaxWorkers set.
	s := NewSpawner(&config.Config{MaxWorkers: 2}, nil, nil)

	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeAgents["a2"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a2"}}
	s.activeMu.Unlock()

	if s.CanSpawn() {
		t.Error("CanSpawn() = true, want false (legacy MaxWorkers=2, 2/2 used)")
	}
}

func TestCapacity(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 5}, nil, nil)

	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeAgents["a2"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a2"}}
	s.activeMu.Unlock()

	cap := s.Capacity()
	if cap.Max != 5 {
		t.Errorf("Max = %d, want 5", cap.Max)
	}
	if cap.Active != 2 {
		t.Errorf("Active = %d, want 2", cap.Active)
	}
	if cap.Available != 3 {
		t.Errorf("Available = %d, want 3", cap.Available)
	}
}

func TestCapacity_AfterRemoval(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 3}, nil, nil)

	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeAgents["a2"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a2"}}
	s.activeMu.Unlock()

	// Simulate agent completion.
	s.activeMu.Lock()
	delete(s.activeAgents, "a1")
	s.activeMu.Unlock()

	cap := s.Capacity()
	if cap.Active != 1 {
		t.Errorf("Active = %d, want 1 after removal", cap.Active)
	}
	if cap.Available != 2 {
		t.Errorf("Available = %d, want 2 after removal", cap.Available)
	}
}

func TestPublishCapacityChanged_NilEventBus(t *testing.T) {
	t.Parallel()
	// Should not panic when eventBus is nil.
	s := NewSpawner(&config.Config{MaxSwarmSize: 3}, nil, nil)
	s.publishCapacityChanged() // no panic = pass
}

type mockEventBus struct {
	events []domain.Event
}

func (m *mockEventBus) Publish(e domain.Event) {
	m.events = append(m.events, e)
}

func (m *mockEventBus) Subscribe() <-chan domain.Event {
	return make(chan domain.Event)
}

func (m *mockEventBus) Unsubscribe(ch <-chan domain.Event) {}

func (m *mockEventBus) ClientCount() int { return 0 }

func TestPublishCapacityChanged_PublishesEvent(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 3}, nil, nil)
	bus := &mockEventBus{}
	s.SetEventBus(bus)

	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeMu.Unlock()

	s.publishCapacityChanged()

	if len(bus.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(bus.events))
	}
	if bus.events[0].Type != domain.EventCapacityChanged {
		t.Errorf("event type = %q, want %q", bus.events[0].Type, domain.EventCapacityChanged)
	}
	cap, ok := bus.events[0].Data.(domain.SwarmCapacity)
	if !ok {
		t.Fatalf("event data type = %T, want domain.SwarmCapacity", bus.events[0].Data)
	}
	if cap.Max != 3 || cap.Active != 1 || cap.Available != 2 {
		t.Errorf("capacity = %+v, want {Max:3 Active:1 Available:2}", cap)
	}
}

func TestSpawnInteractive_AtCapacity(t *testing.T) {
	t.Parallel()
	s := NewSpawner(&config.Config{MaxSwarmSize: 1}, nil, nil)

	// Fill to capacity.
	s.activeMu.Lock()
	s.activeAgents["a1"] = &InteractiveAgent{Info: domain.AgentInfo{ID: "a1"}}
	s.activeMu.Unlock()

	_, err := s.SpawnInteractive(context.Background(), SpawnInteractiveOpts{})
	if !errors.Is(err, ErrAtCapacity) {
		t.Errorf("SpawnInteractive() error = %v, want ErrAtCapacity", err)
	}
}
