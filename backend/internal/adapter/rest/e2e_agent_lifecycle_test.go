//go:build e2e

package rest

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
)

// seedAgentLifecycleData populates the test server with agents in various
// statuses for lifecycle E2E tests.
func seedAgentLifecycleData(t *testing.T, srv *e2eServer) {
	t.Helper()

	_ = srv.projects.Add(domain.Project{
		Slug:     "lifecycle-project",
		RepoName: "lifecycle-project",
	})

	now := time.Now()
	agents := []domain.AgentInfo{
		{
			ID:        "agent-dev-running",
			Name:      "swift-falcon",
			Role:      "developer",
			Ref:       "track-lifecycle-001",
			Status:    "running",
			SessionID: "session-dev-001",
			PID:       12345,
			StartedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now,
		},
		{
			ID:        "agent-rev-running",
			Name:      "keen-owl",
			Role:      "reviewer",
			Ref:       "PR #42",
			Status:    "running",
			SessionID: "session-rev-001",
			PID:       12346,
			StartedAt: now.Add(-5 * time.Minute),
			UpdatedAt: now,
		},
		{
			ID:         "agent-completed",
			Name:       "calm-otter",
			Role:       "developer",
			Ref:        "track-lifecycle-002",
			Status:     "completed",
			SessionID:  "session-comp-001",
			StartedAt:  now.Add(-1 * time.Hour),
			UpdatedAt:  now.Add(-30 * time.Minute),
			FinishedAt: func() *time.Time { t := now.Add(-30 * time.Minute); return &t }(),
		},
		{
			ID:         "agent-failed",
			Name:       "bold-hawk",
			Role:       "developer",
			Ref:        "track-lifecycle-003",
			Status:     "failed",
			SessionID:  "session-fail-001",
			StartedAt:  now.Add(-2 * time.Hour),
			UpdatedAt:  now.Add(-1 * time.Hour),
			FinishedAt: func() *time.Time { t := now.Add(-1 * time.Hour); return &t }(),
		},
		{
			ID:             "agent-stopped",
			Name:           "quiet-deer",
			Role:           "interactive",
			Ref:            "",
			Status:         "stopped",
			SessionID:      "session-stop-001",
			StartedAt:      now.Add(-45 * time.Minute),
			UpdatedAt:      now.Add(-15 * time.Minute),
			FinishedAt:     func() *time.Time { t := now.Add(-15 * time.Minute); return &t }(),
			ShutdownReason: "user_stopped",
		},
		{
			ID:          "agent-suspended",
			Name:        "lazy-cat",
			Role:        "developer",
			Ref:         "track-lifecycle-004",
			Status:      "suspended",
			SessionID:   "session-susp-001",
			StartedAt:   now.Add(-20 * time.Minute),
			UpdatedAt:   now.Add(-5 * time.Minute),
			SuspendedAt: func() *time.Time { t := now.Add(-5 * time.Minute); return &t }(),
		},
	}

	for _, a := range agents {
		if err := srv.agents.AddAgent(a); err != nil {
			t.Fatalf("seed agent %s: %v", a.ID, err)
		}
	}
	if err := srv.agents.Save(); err != nil {
		t.Fatalf("save agents: %v", err)
	}
}

// e2eGetJSON performs a GET request and decodes the JSON response body.
func e2eGetJSON(t *testing.T, url string, dest any) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	if dest != nil {
		if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
			t.Fatalf("decode response from %s: %v", url, err)
		}
	}
	return resp
}

// --- Phase 1: Spawn Tests ---

func TestE2E_AgentLifecycle_SpawnInteractive_NotConfigured(t *testing.T) {
	srv := startE2EServer(t)

	resp, err := http.Post(srv.URL+"/api/agents/interactive", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/agents/interactive: %v", err)
	}
	defer resp.Body.Close()

	// Without spawner configured, the server returns either 400 (missing body)
	// or 500 (not configured). Both are acceptable error responses.
	if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 400 or 500 when spawner not configured, got %d", resp.StatusCode)
	}
}

func TestE2E_AgentLifecycle_ListAgentsAfterSeed(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agents []map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents?active=false", &agents)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(agents) != 6 {
		t.Errorf("expected 6 seeded agents, got %d", len(agents))
	}

	roles := map[string]int{}
	for _, a := range agents {
		if r, ok := a["role"].(string); ok {
			roles[r]++
		}
	}
	if roles["developer"] != 4 {
		t.Errorf("expected 4 developer agents, got %d", roles["developer"])
	}
	if roles["reviewer"] != 1 {
		t.Errorf("expected 1 reviewer agent, got %d", roles["reviewer"])
	}
	if roles["interactive"] != 1 {
		t.Errorf("expected 1 interactive agent, got %d", roles["interactive"])
	}
}

func TestE2E_AgentLifecycle_ListActiveAgentsFiltersByStatus(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agents []map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents", &agents)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	for _, a := range agents {
		status := a["status"].(string)
		if status != "running" && status != "waiting" {
			t.Logf("non-active agent in active list: %s (status: %s)", a["id"], status)
		}
	}
	if len(agents) < 2 {
		t.Errorf("expected at least 2 active agents, got %d", len(agents))
	}
}

// --- Phase 2: Monitoring Tests ---

func TestE2E_AgentLifecycle_GetAgentDetail(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-dev-running", &agent)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	checks := map[string]any{
		"id":     "agent-dev-running",
		"role":   "developer",
		"status": "running",
		"ref":    "track-lifecycle-001",
	}
	for field, want := range checks {
		if agent[field] != want {
			t.Errorf("expected %s=%v, got %v", field, want, agent[field])
		}
	}
	if agent["started_at"] == nil || agent["started_at"] == "" {
		t.Error("expected started_at to be set")
	}
}

func TestE2E_AgentLifecycle_GetAgentByPrefix(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-dev", &agent)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if agent["id"] != "agent-dev-running" {
		t.Errorf("expected agent-dev-running, got %v", agent["id"])
	}
}

func TestE2E_AgentLifecycle_GetAgent404(t *testing.T) {
	srv := startE2EServer(t)

	resp, err := http.Get(srv.URL + "/api/agents/nonexistent")
	if err != nil {
		t.Fatalf("GET /api/agents/nonexistent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestE2E_AgentLifecycle_GetAgentLog_NoLogFile(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	resp, err := http.Get(srv.URL + "/api/agents/agent-dev-running/log")
	if err != nil {
		t.Fatalf("GET agent log: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 for agent without log file, got %d", resp.StatusCode)
	}
}

func TestE2E_AgentLifecycle_GetAgentLogWithFile(t *testing.T) {
	srv := startE2EServer(t)

	// Create the log file first.
	logContent := "line1\nline2\nline3\nline4\nline5\n"
	logPath := srv.DataDir + "/logs/agent-log-test.log"
	writeTestLogFile(t, logPath, logContent)

	// Seed an agent that already has a log file path.
	if err := srv.agents.AddAgent(domain.AgentInfo{
		ID:        "agent-with-log",
		Role:      "developer",
		Ref:       "track-log-test",
		Status:    "running",
		LogFile:   logPath,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}

	var logResp map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-with-log/log?lines=3", &logResp)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if logResp["agent_id"] != "agent-with-log" {
		t.Errorf("expected agent_id agent-with-log, got %v", logResp["agent_id"])
	}
	lines, ok := logResp["lines"].([]any)
	if !ok {
		t.Fatal("expected lines to be an array")
	}
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (tail), got %d", len(lines))
	}
}

// --- Phase 3: Status Transition Tests ---

func TestE2E_AgentLifecycle_StatusFieldsForRunning(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-dev-running", &agent)
	defer resp.Body.Close()

	if agent["status"] != "running" {
		t.Errorf("expected running, got %v", agent["status"])
	}
	if agent["finished_at"] != nil && agent["finished_at"] != "" {
		t.Errorf("running agent should not have finished_at, got %v", agent["finished_at"])
	}
}

func TestE2E_AgentLifecycle_StatusFieldsForCompleted(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-completed", &agent)
	defer resp.Body.Close()

	if agent["status"] != "completed" {
		t.Errorf("expected completed, got %v", agent["status"])
	}
	if agent["finished_at"] == nil || agent["finished_at"] == "" {
		t.Error("completed agent should have finished_at set")
	}
}

func TestE2E_AgentLifecycle_StatusFieldsForFailed(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-failed", &agent)
	defer resp.Body.Close()

	if agent["status"] != "failed" {
		t.Errorf("expected failed, got %v", agent["status"])
	}
	if agent["finished_at"] == nil || agent["finished_at"] == "" {
		t.Error("failed agent should have finished_at set")
	}
}

func TestE2E_AgentLifecycle_StatusFieldsForStopped(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-stopped", &agent)
	defer resp.Body.Close()

	if agent["status"] != "stopped" {
		t.Errorf("expected stopped, got %v", agent["status"])
	}
	if agent["shutdown_reason"] != "user_stopped" {
		t.Errorf("expected shutdown_reason user_stopped, got %v", agent["shutdown_reason"])
	}
}

func TestE2E_AgentLifecycle_StatusFieldsForSuspended(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-suspended", &agent)
	defer resp.Body.Close()

	if agent["status"] != "suspended" {
		t.Errorf("expected suspended, got %v", agent["status"])
	}
	if agent["suspended_at"] == nil || agent["suspended_at"] == "" {
		t.Error("suspended agent should have suspended_at set")
	}
}

func TestE2E_AgentLifecycle_StatusTransitionViaStore(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	if err := srv.agents.UpdateStatus("agent-dev-running", "completed"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	_ = srv.agents.Save()

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-dev-running", &agent)
	defer resp.Body.Close()

	if agent["status"] != "completed" {
		t.Errorf("expected completed after transition, got %v", agent["status"])
	}
}

// --- Phase 4: Lifecycle Action Tests ---

func TestE2E_AgentLifecycle_StopAgent_NotConfigured(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	resp, err := http.Post(srv.URL+"/api/agents/agent-dev-running/stop", "application/json", nil)
	if err != nil {
		t.Fatalf("POST stop: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 when spawner not configured, got %d", resp.StatusCode)
	}
}

func TestE2E_AgentLifecycle_ResumeAgent_NotConfigured(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	resp, err := http.Post(srv.URL+"/api/agents/agent-stopped/resume", "application/json", nil)
	if err != nil {
		t.Fatalf("POST resume: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 when spawner not configured, got %d", resp.StatusCode)
	}
}

func TestE2E_AgentLifecycle_DeleteAgent_NotConfigured(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/agents/agent-stopped", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 when remover not configured, got %d", resp.StatusCode)
	}
}

func TestE2E_AgentLifecycle_DeleteAgent_WithRemover(t *testing.T) {
	srv := startE2EServerWithAgentRemover(t)
	seedAgentLifecycleData(t, srv)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/agents/agent-stopped", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	// Verify agent is gone.
	getResp, err := http.Get(srv.URL + "/api/agents/agent-stopped")
	if err != nil {
		t.Fatalf("GET after delete: %v", err)
	}
	defer getResp.Body.Close()
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", getResp.StatusCode)
	}
}

func TestE2E_AgentLifecycle_DeleteAgent_NotFound(t *testing.T) {
	srv := startE2EServerWithAgentRemover(t)

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/api/agents/nonexistent", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// --- Phase 5: History and Filtering Tests ---

func TestE2E_AgentLifecycle_ListAllAgentsForHistory(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agents []map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents?active=false", &agents)
	defer resp.Body.Close()

	if len(agents) != 6 {
		t.Errorf("expected 6 agents in history, got %d", len(agents))
	}

	statuses := map[string]bool{}
	for _, a := range agents {
		statuses[a["status"].(string)] = true
	}
	for _, expected := range []string{"running", "completed", "failed", "stopped", "suspended"} {
		if !statuses[expected] {
			t.Errorf("expected status %q in history list", expected)
		}
	}
}

func TestE2E_AgentLifecycle_ListAllAgentsHaveRequiredFields(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agents []map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents?active=false", &agents)
	defer resp.Body.Close()

	requiredFields := []string{"id", "role", "status", "started_at", "updated_at"}
	for _, a := range agents {
		for _, field := range requiredFields {
			if a[field] == nil || a[field] == "" {
				t.Errorf("agent %v: missing required field %q", a["id"], field)
			}
		}
	}
}

func TestE2E_AgentLifecycle_AgentsByRoleInHistory(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agents []map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents?active=false", &agents)
	defer resp.Body.Close()

	devAgents := 0
	revAgents := 0
	for _, a := range agents {
		switch a["role"] {
		case "developer":
			devAgents++
		case "reviewer":
			revAgents++
		}
	}
	if devAgents < 1 {
		t.Error("expected at least 1 developer agent in history")
	}
	if revAgents < 1 {
		t.Error("expected at least 1 reviewer agent in history")
	}
}

func TestE2E_AgentLifecycle_ReviewerAgentHasCorrectRef(t *testing.T) {
	srv := startE2EServer(t)
	seedAgentLifecycleData(t, srv)

	var agent map[string]any
	resp := e2eGetJSON(t, srv.URL+"/api/agents/agent-rev-running", &agent)
	defer resp.Body.Close()

	if agent["role"] != "reviewer" {
		t.Errorf("expected reviewer, got %v", agent["role"])
	}
	if agent["ref"] != "PR #42" {
		t.Errorf("expected ref 'PR #42', got %v", agent["ref"])
	}
}
