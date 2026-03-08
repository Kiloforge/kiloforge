//go:build integration

package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"kiloforge/internal/adapter/badge"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/jsonfile"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
)

// testServer wraps an HTTP server for integration tests.
type testServer struct {
	URL    string
	cancel context.CancelFunc
}

// startTestServer creates a fully wired server on a random port.
func startTestServer(t *testing.T) *testServer {
	t.Helper()

	dir := t.TempDir()
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dir,
		GiteaAdminUser: "kiloforger",
	}
	reg := &jsonfile.ProjectStore{
		Version:  1,
		Projects: map[string]domain.Project{},
	}
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	store := &jsonfile.AgentStore{}
	prTracker := sqlite.NewPRTrackingStore(db)

	// Find a random available port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	lockMgr := lock.New(dir)
	lockMgr.StartReaper(ctx)

	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:     store,
		LockMgr:    lockMgr,
		ProjectDir: dir,
		GiteaURL:   cfg.GiteaURL(),
		Projects:   len(reg.Projects),
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	// Webhook route.
	srv := NewServer(cfg, reg, store, prTracker, port)
	mux.HandleFunc("/webhook", srv.handleWebhook)

	// Badge routes.
	prLoader := func(slug string) (*domain.PRTracking, error) { return nil, nil }
	badgeHandler := badge.NewHandler(store, prLoader)
	badgeHandler.RegisterRoutes(mux)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		httpSrv.Shutdown(context.Background())
	}()

	go func() {
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Wait for server to be ready.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(url + "/health"); err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Cleanup(cancel)
	return &testServer{URL: url, cancel: cancel}
}

func TestIntegration_Health(t *testing.T) {
	srv := startTestServer(t)

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health: got %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("health status: got %v, want 'ok'", body["status"])
	}
}

func TestIntegration_Agents(t *testing.T) {
	srv := startTestServer(t)

	resp, err := http.Get(srv.URL + "/api/agents")
	if err != nil {
		t.Fatalf("GET /api/agents: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/agents: got %d, want 200", resp.StatusCode)
	}

	var agents []any
	json.NewDecoder(resp.Body).Decode(&agents)
	if len(agents) != 0 {
		t.Errorf("expected empty agents list, got %d", len(agents))
	}
}

func TestIntegration_Status(t *testing.T) {
	srv := startTestServer(t)

	resp, err := http.Get(srv.URL + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/status: got %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if _, ok := body["total_agents"]; !ok {
		t.Error("status response missing 'total_agents' field")
	}
}

func TestIntegration_Locks(t *testing.T) {
	srv := startTestServer(t)

	resp, err := http.Get(srv.URL + "/api/locks")
	if err != nil {
		t.Fatalf("GET /api/locks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/locks: got %d, want 200", resp.StatusCode)
	}

	var locks []any
	json.NewDecoder(resp.Body).Decode(&locks)
	if len(locks) != 0 {
		t.Errorf("expected empty locks list, got %d", len(locks))
	}
}

func TestIntegration_BadgeEndpoint(t *testing.T) {
	srv := startTestServer(t)

	resp, err := http.Get(srv.URL + "/api/badges/track/nonexistent")
	if err != nil {
		t.Fatalf("GET badge: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("badge: got %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("badge content-type: got %q, want %q", ct, "image/svg+xml")
	}
}

func TestIntegration_Quota(t *testing.T) {
	srv := startTestServer(t)

	resp, err := http.Get(srv.URL + "/api/quota")
	if err != nil {
		t.Fatalf("GET /api/quota: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/quota: got %d, want 200", resp.StatusCode)
	}
}
