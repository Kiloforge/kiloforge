//go:build e2e

package rest

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestE2E_HealthCheck(t *testing.T) {
	srv := startE2EServer(t)

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: got %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health status: got %v, want 'ok'", body["status"])
	}
}

func TestE2E_HealthCheck_MockAgentBuilds(t *testing.T) {
	srv := startE2EServer(t)

	// Verify mock agent binary was built.
	if srv.MockAgentBin == "" {
		t.Fatal("mock agent binary path is empty")
	}
}

func TestE2E_SeedAndListAgents(t *testing.T) {
	srv := startE2EServer(t)
	seedTestData(t, srv)

	// Use active=false to include all agents (including completed).
	resp, err := http.Get(srv.URL + "/api/agents?active=false")
	if err != nil {
		t.Fatalf("GET /api/agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/agents: got %d, want 200", resp.StatusCode)
	}

	var agents []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Fatalf("decode agents: %v", err)
	}
	if len(agents) < 2 {
		t.Errorf("expected at least 2 seeded agents, got %d", len(agents))
	}
}

func TestE2E_CleanupResetsData(t *testing.T) {
	srv := startE2EServer(t)
	seedTestData(t, srv)
	cleanupTestData(t, srv)

	resp, err := http.Get(srv.URL + "/api/agents")
	if err != nil {
		t.Fatalf("GET /api/agents: %v", err)
	}
	defer resp.Body.Close()

	var agents []any
	json.NewDecoder(resp.Body).Decode(&agents)
	if len(agents) != 0 {
		t.Errorf("expected 0 agents after cleanup, got %d", len(agents))
	}
}
