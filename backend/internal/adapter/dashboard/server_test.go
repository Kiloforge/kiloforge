package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

// testAgentLister is an in-memory AgentLister for tests.
type testAgentLister struct {
	agents []domain.AgentInfo
}

func (t *testAgentLister) Agents() []domain.AgentInfo { return t.agents }
func (t *testAgentLister) Load() error                { return nil }
func (t *testAgentLister) FindAgent(id string) (*domain.AgentInfo, error) {
	for i := range t.agents {
		if t.agents[i].ID == id {
			return &t.agents[i], nil
		}
	}
	return nil, domain.ErrAgentNotFound
}

// testProjectLister is an in-memory ProjectLister for tests.
type testProjectLister struct {
	projects []domain.Project
}

func (t *testProjectLister) List() []domain.Project { return t.projects }

func TestSSEHub_BroadcastAndSubscribe(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()

	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	if hub.ClientCount() != 1 {
		t.Fatalf("client count = %d, want 1", hub.ClientCount())
	}

	hub.Broadcast(SSEEvent{Type: "test", Data: "hello"})

	select {
	case event := <-ch:
		if event.Type != "test" {
			t.Errorf("event type = %q, want test", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestSSEHub_Unsubscribe(t *testing.T) {
	t.Parallel()
	hub := NewSSEHub()

	ch := hub.Subscribe()
	hub.Unsubscribe(ch)

	if hub.ClientCount() != 0 {
		t.Errorf("client count = %d, want 0", hub.ClientCount())
	}
}

func TestRegisterNonAPIRoutes_MountsOnExternalMux(t *testing.T) {
	t.Parallel()
	s := New(0, &testAgentLister{}, nil, "http://localhost:3000", &testProjectLister{})

	externalMux := http.NewServeMux()
	s.RegisterNonAPIRoutes(externalMux)

	// SPA static route should be registered and serve the frontend.
	req := httptest.NewRequest("GET", "/-/", nil)
	w := httptest.NewRecorder()
	externalMux.ServeHTTP(w, req)
	if w.Code == http.StatusNotFound {
		t.Errorf("/-/: got 404, expected SPA route to be registered")
	}
}
