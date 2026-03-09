//go:build e2e

package rest

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kiloforge/internal/adapter/badge"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// e2eServer wraps an HTTP server for E2E tests.
type e2eServer struct {
	URL          string
	MockAgentBin string
	DataDir      string
	cancel       context.CancelFunc
	db           *sql.DB
	projects     *sqlite.ProjectStore
	agents       port.AgentStore
}

// startE2EServer builds the mock agent binary, creates a temp SQLite DB,
// boots a fully wired Fiber server on a random port, and returns the
// server URL plus cleanup functions.
func startE2EServer(t *testing.T) *e2eServer {
	t.Helper()

	// Build mock agent binary to temp directory.
	mockBin := buildMockAgentBinary(t)

	dir := t.TempDir()
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dir,
		GiteaAdminUser: "kiloforger",
	}
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	reg := sqlite.NewProjectStore(db)
	store := sqlite.NewAgentStore(db)
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
		Agents:   store,
		LockMgr:  lockMgr,
		Projects: reg,
		GiteaURL: cfg.GiteaURL(),
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
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(url + "/health"); err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Cleanup(cancel)

	return &e2eServer{
		URL:          url,
		MockAgentBin: mockBin,
		DataDir:      dir,
		cancel:       cancel,
		db:           db,
		projects:     reg,
		agents:       store,
	}
}

// seedTestData populates the test server with sample data for E2E tests.
func seedTestData(t *testing.T, srv *e2eServer) {
	t.Helper()

	// Create a test project.
	err := srv.projects.Add(domain.Project{
		Slug:     "test-project",
		RepoName: "test-project",
	})
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}

	// Add sample agents with various statuses.
	agents := []domain.AgentInfo{
		{
			ID:        "agent-running-1",
			Name:      "swift-falcon",
			Role:      "developer",
			Ref:       "track-001",
			Status:    "running",
			StartedAt: time.Now().Add(-10 * time.Minute),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "agent-completed-1",
			Name:      "calm-otter",
			Role:      "reviewer",
			Ref:       "PR #1",
			Status:    "completed",
			StartedAt: time.Now().Add(-30 * time.Minute),
			UpdatedAt: time.Now().Add(-5 * time.Minute),
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

// cleanupTestData resets the database by removing all agents.
func cleanupTestData(t *testing.T, srv *e2eServer) {
	t.Helper()
	for _, a := range srv.agents.Agents() {
		if err := srv.agents.RemoveAgent(a.ID); err != nil {
			t.Logf("cleanup agent %s: %v", a.ID, err)
		}
	}
	_ = srv.agents.Save()
}

// buildMockAgentBinary builds the mock-agent Go binary and returns the path.
func buildMockAgentBinary(t *testing.T) string {
	t.Helper()

	binDir := t.TempDir()
	bin := filepath.Join(binDir, "mock-agent")

	// Resolve the mock-agent source directory.
	mockSrc := findMockAgentSource(t)

	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = mockSrc

	// Worktree-safe VCS env.
	gitCommon, _ := exec.Command("git", "rev-parse", "--git-common-dir").CombinedOutput()
	gitTop, _ := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
	cmd.Env = append(os.Environ(),
		"GOWORK=off",
		"GIT_DIR="+strings.TrimSpace(string(gitCommon)),
		"GIT_WORK_TREE="+strings.TrimSpace(string(gitTop)),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build mock-agent: %v\n%s", err, out)
	}
	return bin
}

// findMockAgentSource locates the mock-agent source directory relative to the test file.
func findMockAgentSource(t *testing.T) string {
	t.Helper()
	// We're in backend/internal/adapter/rest/, mock-agent is at backend/internal/adapter/agent/testdata/mock-agent/
	candidates := []string{
		"../agent/testdata/mock-agent",
		filepath.Join("..", "agent", "testdata", "mock-agent"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "main.go")); err == nil {
			return c
		}
	}
	t.Fatal("cannot find mock-agent source directory")
	return ""
}
