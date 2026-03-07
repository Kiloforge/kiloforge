package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"crelay/internal/agent"
	"crelay/internal/core/domain"
)

// testAgentLister is an in-memory AgentLister for tests.
type testAgentLister struct {
	agents []domain.AgentInfo
}

func (t *testAgentLister) Agents() []domain.AgentInfo        { return t.agents }
func (t *testAgentLister) Load() error                       { return nil }
func (t *testAgentLister) FindAgent(id string) (*domain.AgentInfo, error) {
	for i := range t.agents {
		if t.agents[i].ID == id {
			return &t.agents[i], nil
		}
	}
	return nil, domain.ErrAgentNotFound
}

// testQuotaReader is a test QuotaReader.
type testQuotaReader struct {
	agentUsage  map[string]*agent.AgentUsage
	totalUsage  agent.TotalUsage
	rateLimited bool
	retryAfter  time.Duration
}

func (t *testQuotaReader) GetAgentUsage(id string) *agent.AgentUsage {
	if t.agentUsage == nil {
		return nil
	}
	return t.agentUsage[id]
}
func (t *testQuotaReader) GetTotalUsage() agent.TotalUsage { return t.totalUsage }
func (t *testQuotaReader) IsRateLimited() bool             { return t.rateLimited }
func (t *testQuotaReader) RetryAfter() time.Duration       { return t.retryAfter }

func newTestServer(agents AgentLister, quota QuotaReader, projectDir string) *Server {
	s := &Server{
		port:       0,
		agents:     agents,
		quota:      quota,
		giteaURL:   "http://localhost:3000",
		projectDir: projectDir,
		hub:        NewSSEHub(),
		mux:        http.NewServeMux(),
	}
	s.routes()
	return s
}

func TestHandleAgents_Empty(t *testing.T) {
	t.Parallel()
	s := newTestServer(&testAgentLister{}, nil, t.TempDir())

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result []any
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 0 {
		t.Errorf("expected empty array, got %d items", len(result))
	}
}

func TestHandleAgents_WithAgents(t *testing.T) {
	t.Parallel()
	agents := &testAgentLister{
		agents: []domain.AgentInfo{
			{ID: "dev-1", Role: "developer", Status: "running", StartedAt: time.Now().Add(-5 * time.Minute)},
			{ID: "rev-1", Role: "reviewer", Status: "completed"},
		},
	}
	quota := &testQuotaReader{
		agentUsage: map[string]*agent.AgentUsage{
			"dev-1": {AgentID: "dev-1", TotalCostUSD: 0.42, InputTokens: 1000, OutputTokens: 500},
		},
	}
	s := newTestServer(agents, quota, t.TempDir())

	req := httptest.NewRequest("GET", "/api/agents", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result []map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(result))
	}
	if result[0]["id"] != "dev-1" {
		t.Errorf("first agent id = %v, want dev-1", result[0]["id"])
	}
	if result[0]["cost_usd"] != 0.42 {
		t.Errorf("cost_usd = %v, want 0.42", result[0]["cost_usd"])
	}
}

func TestHandleAgent_NotFound(t *testing.T) {
	t.Parallel()
	s := newTestServer(&testAgentLister{}, nil, t.TempDir())

	req := httptest.NewRequest("GET", "/api/agents/nonexistent", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleAgent_Found(t *testing.T) {
	t.Parallel()
	agents := &testAgentLister{
		agents: []domain.AgentInfo{
			{ID: "dev-1", Role: "developer", Status: "running"},
		},
	}
	s := newTestServer(agents, nil, t.TempDir())

	req := httptest.NewRequest("GET", "/api/agents/dev-1", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["id"] != "dev-1" {
		t.Errorf("id = %v, want dev-1", result["id"])
	}
}

func TestHandleQuota_NoTracker(t *testing.T) {
	t.Parallel()
	s := newTestServer(&testAgentLister{}, nil, t.TempDir())

	req := httptest.NewRequest("GET", "/api/quota", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["rate_limited"] != false {
		t.Errorf("rate_limited = %v, want false", result["rate_limited"])
	}
}

func TestHandleQuota_RateLimited(t *testing.T) {
	t.Parallel()
	quota := &testQuotaReader{
		totalUsage:  agent.TotalUsage{TotalCostUSD: 1.50, InputTokens: 5000, OutputTokens: 2000, AgentCount: 2},
		rateLimited: true,
		retryAfter:  30 * time.Second,
	}
	s := newTestServer(&testAgentLister{}, quota, t.TempDir())

	req := httptest.NewRequest("GET", "/api/quota", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["rate_limited"] != true {
		t.Errorf("rate_limited = %v, want true", result["rate_limited"])
	}
	if result["total_cost_usd"] != 1.5 {
		t.Errorf("total_cost_usd = %v, want 1.5", result["total_cost_usd"])
	}
}

func TestHandleTracks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Create tracks.md in the expected location.
	conductorDir := filepath.Join(dir, ".agent", "conductor")
	os.MkdirAll(conductorDir, 0o755)
	os.WriteFile(filepath.Join(conductorDir, "tracks.md"), []byte(`# Tracks

| Status | Track ID | Title | Created | Updated |
| ------ | -------- | ----- | ------- | ------- |
| [x] | track-1 | First Track | 2026-03-01 | 2026-03-01 |
| [ ] | track-2 | Second Track | 2026-03-02 | 2026-03-02 |
`), 0o644)

	s := newTestServer(&testAgentLister{}, nil, dir)

	req := httptest.NewRequest("GET", "/api/tracks", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result []map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)
	if len(result) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(result))
	}
	if result[0]["status"] != "complete" {
		t.Errorf("first track status = %v, want complete", result[0]["status"])
	}
	if result[1]["status"] != "pending" {
		t.Errorf("second track status = %v, want pending", result[1]["status"])
	}
}

func TestHandleTracks_NoFile(t *testing.T) {
	t.Parallel()
	s := newTestServer(&testAgentLister{}, nil, t.TempDir())

	req := httptest.NewRequest("GET", "/api/tracks", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
}

func TestHandleStatus(t *testing.T) {
	t.Parallel()
	agents := &testAgentLister{
		agents: []domain.AgentInfo{
			{ID: "dev-1", Status: "running"},
			{ID: "dev-2", Status: "running"},
			{ID: "rev-1", Status: "completed"},
		},
	}
	quota := &testQuotaReader{
		totalUsage: agent.TotalUsage{TotalCostUSD: 0.75},
	}
	s := newTestServer(agents, quota, t.TempDir())

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	if result["total_agents"].(float64) != 3 {
		t.Errorf("total_agents = %v, want 3", result["total_agents"])
	}
	if result["total_cost_usd"].(float64) != 0.75 {
		t.Errorf("total_cost_usd = %v, want 0.75", result["total_cost_usd"])
	}
	counts := result["agent_counts"].(map[string]any)
	if counts["running"].(float64) != 2 {
		t.Errorf("running count = %v, want 2", counts["running"])
	}
}

func TestHandleAgentLog_NoLogFile(t *testing.T) {
	t.Parallel()
	agents := &testAgentLister{
		agents: []domain.AgentInfo{
			{ID: "dev-1", Status: "running"},
		},
	}
	s := newTestServer(agents, nil, t.TempDir())

	req := httptest.NewRequest("GET", "/api/agents/dev-1/log", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestHandleAgentLog_WithLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logFile := filepath.Join(dir, "agent.log")
	os.WriteFile(logFile, []byte("line1\nline2\nline3\nline4\nline5\n"), 0o644)

	agents := &testAgentLister{
		agents: []domain.AgentInfo{
			{ID: "dev-1", Status: "running", LogFile: logFile},
		},
	}
	s := newTestServer(agents, nil, t.TempDir())

	req := httptest.NewRequest("GET", "/api/agents/dev-1/log?lines=3", nil)
	w := httptest.NewRecorder()
	s.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var result map[string]any
	json.Unmarshal(w.Body.Bytes(), &result)
	lines := result["lines"].([]any)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line3" {
		t.Errorf("first line = %v, want line3", lines[0])
	}
}

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
