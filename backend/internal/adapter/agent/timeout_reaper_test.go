package agent

import (
	"context"
	"testing"
	"time"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

type fakeAgentStore struct {
	agents    []domain.AgentInfo
	halted    []string
	statusLog map[string]string
}

func newFakeStore(agents ...domain.AgentInfo) *fakeAgentStore {
	return &fakeAgentStore{agents: agents, statusLog: make(map[string]string)}
}

func (s *fakeAgentStore) Load() error                                   { return nil }
func (s *fakeAgentStore) Save() error                                   { return nil }
func (s *fakeAgentStore) AddAgent(_ domain.AgentInfo) error             { return nil }
func (s *fakeAgentStore) FindByRef(_ string) *domain.AgentInfo          { return nil }
func (s *fakeAgentStore) RemoveAgent(_ string) error                    { return nil }
func (s *fakeAgentStore) AgentsByStatus(_ ...string) []domain.AgentInfo { return nil }

func (s *fakeAgentStore) FindAgent(idPrefix string) (*domain.AgentInfo, error) {
	for i := range s.agents {
		if s.agents[i].ID == idPrefix {
			return &s.agents[i], nil
		}
	}
	return nil, nil
}

func (s *fakeAgentStore) Agents() []domain.AgentInfo { return s.agents }

func (s *fakeAgentStore) UpdateStatus(idPrefix, status string) error {
	s.statusLog[idPrefix] = status
	for i := range s.agents {
		if s.agents[i].ID == idPrefix {
			s.agents[i].Status = status
		}
	}
	return nil
}

func (s *fakeAgentStore) HaltAgent(idPrefix string) error {
	s.halted = append(s.halted, idPrefix)
	return nil
}

type fakeEventBus struct{ events []domain.Event }

func (b *fakeEventBus) Publish(e domain.Event)       { b.events = append(b.events, e) }
func (b *fakeEventBus) Subscribe() <-chan domain.Event { return make(chan domain.Event) }
func (b *fakeEventBus) Unsubscribe(_ <-chan domain.Event) {}
func (b *fakeEventBus) ClientCount() int              { return 0 }

var _ port.AgentStore = (*fakeAgentStore)(nil)
var _ port.EventBus = (*fakeEventBus)(nil)

func TestTimeoutReaper_ReapsExpiredAgents(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		domain.AgentInfo{ID: "dev-1", Role: "developer", Status: "running", StartedAt: time.Now().Add(-3 * time.Hour)},
		domain.AgentInfo{ID: "dev-2", Role: "developer", Status: "running", StartedAt: time.Now().Add(-30 * time.Minute)},
	)
	bus := &fakeEventBus{}
	reaper := NewTimeoutReaper(store, &config.Config{AgentMaxDuration: "2h"}, bus)
	reaper.reap()

	if len(store.halted) != 1 || store.halted[0] != "dev-1" {
		t.Errorf("expected dev-1 halted, got %v", store.halted)
	}
	if store.statusLog["dev-1"] != "force-killed" {
		t.Errorf("expected force-killed, got %q", store.statusLog["dev-1"])
	}
	if _, ok := store.statusLog["dev-2"]; ok {
		t.Error("dev-2 should not be reaped")
	}
	if len(bus.events) != 1 {
		t.Errorf("expected 1 event, got %d", len(bus.events))
	}
}

func TestTimeoutReaper_SkipsInteractiveAgents(t *testing.T) {
	t.Parallel()
	store := newFakeStore(domain.AgentInfo{ID: "i-1", Role: "interactive", Status: "running", StartedAt: time.Now().Add(-5 * time.Hour)})
	reaper := NewTimeoutReaper(store, &config.Config{AgentMaxDuration: "2h"}, &fakeEventBus{})
	reaper.reap()
	if len(store.halted) != 0 {
		t.Errorf("interactive should not be halted, got %v", store.halted)
	}
}

func TestTimeoutReaper_SkipsTerminalAgents(t *testing.T) {
	t.Parallel()
	store := newFakeStore(domain.AgentInfo{ID: "d-1", Role: "developer", Status: "completed", StartedAt: time.Now().Add(-5 * time.Hour)})
	reaper := NewTimeoutReaper(store, &config.Config{AgentMaxDuration: "2h"}, &fakeEventBus{})
	reaper.reap()
	if len(store.halted) != 0 {
		t.Errorf("terminal should not be halted, got %v", store.halted)
	}
}

func TestTimeoutReaper_DisabledWhenZero(t *testing.T) {
	t.Parallel()
	store := newFakeStore(domain.AgentInfo{ID: "d-1", Role: "developer", Status: "running", StartedAt: time.Now().Add(-100 * time.Hour)})
	reaper := NewTimeoutReaper(store, &config.Config{AgentMaxDuration: "0s"}, &fakeEventBus{})
	reaper.reap()
	if len(store.halted) != 0 {
		t.Errorf("disabled timeout should not halt, got %v", store.halted)
	}
}

func TestTimeoutReaper_ReapsMultipleRoles(t *testing.T) {
	t.Parallel()
	store := newFakeStore(
		domain.AgentInfo{ID: "r-1", Role: "reviewer", Status: "running", StartedAt: time.Now().Add(-3 * time.Hour)},
		domain.AgentInfo{ID: "a-1", Role: "architect", Status: "waiting", StartedAt: time.Now().Add(-3 * time.Hour)},
	)
	reaper := NewTimeoutReaper(store, &config.Config{AgentMaxDuration: "2h"}, &fakeEventBus{})
	reaper.reap()
	if len(store.halted) != 2 {
		t.Errorf("expected 2 halted, got %d: %v", len(store.halted), store.halted)
	}
}

func TestTimeoutReaper_Start_CancelStops(t *testing.T) {
	t.Parallel()
	reaper := NewTimeoutReaper(newFakeStore(), &config.Config{AgentMaxDuration: "2h"}, &fakeEventBus{})
	ctx, cancel := context.WithCancel(context.Background())
	reaper.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()
	time.Sleep(50 * time.Millisecond)
}

func TestTimeoutReaper_NilEventBus(t *testing.T) {
	t.Parallel()
	store := newFakeStore(domain.AgentInfo{ID: "d-1", Role: "developer", Status: "running", StartedAt: time.Now().Add(-3 * time.Hour)})
	reaper := NewTimeoutReaper(store, &config.Config{AgentMaxDuration: "2h"}, nil)
	reaper.reap()
	if len(store.halted) != 1 {
		t.Errorf("expected 1 halted, got %d", len(store.halted))
	}
}
