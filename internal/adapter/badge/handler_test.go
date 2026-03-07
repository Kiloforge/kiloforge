package badge

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"crelay/internal/core/domain"
)

// mockAgentFinder implements AgentFinder for tests.
type mockAgentFinder struct {
	agents []domain.AgentInfo
}

func (m *mockAgentFinder) Agents() []domain.AgentInfo { return m.agents }
func (m *mockAgentFinder) Load() error                { return nil }
func (m *mockAgentFinder) FindAgent(id string) (*domain.AgentInfo, error) {
	for i := range m.agents {
		if m.agents[i].ID == id || strings.HasPrefix(m.agents[i].ID, id) {
			return &m.agents[i], nil
		}
	}
	return nil, domain.ErrAgentNotFound
}

func setupBadgeHandler(agents []domain.AgentInfo, prLoader PRTrackingLoader) (*Handler, *http.ServeMux) {
	af := &mockAgentFinder{agents: agents}
	h := NewHandler(af, prLoader)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return h, mux
}

func TestTrackBadge_RunningAgent(t *testing.T) {
	_, mux := setupBadgeHandler([]domain.AgentInfo{
		{ID: "a1", Role: "developer", Ref: "my-track_123", Status: "running", StartedAt: time.Now()},
	}, nil)

	req := httptest.NewRequest("GET", "/api/badges/track/my-track_123", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Errorf("expected image/svg+xml, got %s", ct)
	}
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Errorf("expected no-cache in Cache-Control, got %s", cc)
	}
	body := w.Body.String()
	if !strings.Contains(body, "running") {
		t.Error("badge should contain 'running' status")
	}
	if !strings.Contains(body, "#4c1") {
		t.Error("badge should have green color for running")
	}
	if err := xml.Unmarshal(w.Body.Bytes(), new(any)); err != nil {
		t.Errorf("invalid SVG XML: %v", err)
	}
}

func TestTrackBadge_Pending(t *testing.T) {
	_, mux := setupBadgeHandler(nil, nil)

	req := httptest.NewRequest("GET", "/api/badges/track/unknown-track", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "pending") {
		t.Error("badge should show 'pending' for unknown track")
	}
}

func TestPRBadge_BothAgents(t *testing.T) {
	loader := func(slug string) (*domain.PRTracking, error) {
		return &domain.PRTracking{
			PRNumber:         5,
			DeveloperAgentID: "dev-1",
			ReviewerAgentID:  "rev-1",
		}, nil
	}
	_, mux := setupBadgeHandler([]domain.AgentInfo{
		{ID: "dev-1", Role: "developer", Status: "running"},
		{ID: "rev-1", Role: "reviewer", Status: "waiting"},
	}, loader)

	req := httptest.NewRequest("GET", "/api/badges/pr/myproj/5", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "dev: running") {
		t.Error("badge should contain dev status")
	}
	if !strings.Contains(body, "rev: waiting") {
		t.Error("badge should contain rev status")
	}
}

func TestPRBadge_NoTracking(t *testing.T) {
	_, mux := setupBadgeHandler(nil, nil)

	req := httptest.NewRequest("GET", "/api/badges/pr/myproj/5", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "unknown") {
		t.Error("badge should show 'unknown' when no PR loader")
	}
}

func TestAgentBadge_Found(t *testing.T) {
	_, mux := setupBadgeHandler([]domain.AgentInfo{
		{ID: "agent-abc", Role: "reviewer", Status: "completed"},
	}, nil)

	req := httptest.NewRequest("GET", "/api/badges/agent/agent-abc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "reviewer") {
		t.Error("badge should show agent role")
	}
	if !strings.Contains(body, "completed") {
		t.Error("badge should show agent status")
	}
}

func TestAgentBadge_NotFound(t *testing.T) {
	_, mux := setupBadgeHandler(nil, nil)

	req := httptest.NewRequest("GET", "/api/badges/agent/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "unknown") {
		t.Error("badge should show 'unknown' for missing agent")
	}
}

func TestTrackBadge_PicksMostRecent(t *testing.T) {
	old := time.Now().Add(-1 * time.Hour)
	recent := time.Now()
	_, mux := setupBadgeHandler([]domain.AgentInfo{
		{ID: "a1", Ref: "track-x", Status: "failed", StartedAt: old},
		{ID: "a2", Ref: "track-x", Status: "running", StartedAt: recent},
	}, nil)

	req := httptest.NewRequest("GET", "/api/badges/track/track-x", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "running") {
		t.Error("should pick most recent agent (running), not old one (failed)")
	}
}
