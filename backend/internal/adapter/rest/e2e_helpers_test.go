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
	"sync"
	"testing"
	"time"

	"kiloforge/internal/adapter/badge"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/internal/core/service"
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
	boardStore   port.BoardStore
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
		DataDir: dir,
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

	projectMgr := newE2EProjectManager(reg)

	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:     store,
		LockMgr:    lockMgr,
		Projects:   reg,
		ProjectMgr: projectMgr,
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	// Webhook route.
	srv := NewServer(cfg, reg, store, prTracker, "127.0.0.1", port)
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

// startE2EServerWithAgentRemover is like startE2EServer but wires up
// the AgentRemover so DELETE /api/agents/{id} works.
func startE2EServerWithAgentRemover(t *testing.T) *e2eServer {
	t.Helper()

	mockBin := buildMockAgentBinary(t)
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir: dir,
	}
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	reg := sqlite.NewProjectStore(db)
	store := sqlite.NewAgentStore(db)
	prTracker := sqlite.NewPRTrackingStore(db)

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

	projectMgr := newE2EProjectManager(reg)

	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:       store,
		LockMgr:      lockMgr,
		Projects:     reg,
		ProjectMgr:   projectMgr,
		AgentRemover: store, // Wire up remover for delete tests.
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	srv := NewServer(cfg, reg, store, prTracker, "127.0.0.1", port)
	mux.HandleFunc("/webhook", srv.handleWebhook)

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

// startE2EServerWithBoard is like startE2EServer but wires up
// the BoardService so board API endpoints work.
func startE2EServerWithBoard(t *testing.T) *e2eServer {
	t.Helper()

	mockBin := buildMockAgentBinary(t)
	dir := t.TempDir()
	cfg := &config.Config{
		DataDir: dir,
	}
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	reg := sqlite.NewProjectStore(db)
	store := sqlite.NewAgentStore(db)
	prTracker := sqlite.NewPRTrackingStore(db)
	boardStore := sqlite.NewBoardStore(db)
	boardSvc := service.NewNativeBoardService(boardStore)

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

	projectMgr := newE2EProjectManager(reg)

	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:      store,
		LockMgr:     lockMgr,
		Projects:    reg,
		ProjectMgr:  projectMgr,
		BoardSvc:    boardSvc,
		TrackReader: e2eTrackReader{},
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	srv := NewServer(cfg, reg, store, prTracker, "127.0.0.1", port)
	mux.HandleFunc("/webhook", srv.handleWebhook)

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
		boardStore:   boardStore,
	}
}

// writeTestLogFile creates a log file at the given path with the given content.
func writeTestLogFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir for log: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
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
			Role:      "developer",
			Ref:       "track-completed-001",
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

// e2eProjectManager is an in-memory ProjectManager for E2E tests.
// It stores projects in the SQLite project store without needing a real Gitea instance.
type e2eProjectManager struct {
	mu    sync.Mutex
	store *sqlite.ProjectStore
}

func newE2EProjectManager(store *sqlite.ProjectStore) *e2eProjectManager {
	return &e2eProjectManager{store: store}
}

func (m *e2eProjectManager) AddProject(_ context.Context, remoteURL, name string, opts ...domain.AddProjectOpts) (*domain.AddProjectResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Derive slug from URL or use provided name.
	slug := name
	if slug == "" {
		slug = deriveSlug(remoteURL)
	}
	if slug == "" {
		return nil, fmt.Errorf("cannot derive project name from URL")
	}

	// Check for duplicates.
	if _, ok := m.store.FindByRepoName(slug); ok {
		return nil, domain.ErrProjectExists
	}

	p := domain.Project{
		Slug:         slug,
		RepoName:     slug,
		OriginRemote: remoteURL,
		Active:       true,
		RegisteredAt: time.Now(),
	}
	if err := m.store.Add(p); err != nil {
		return nil, err
	}
	return &domain.AddProjectResult{Project: p}, nil
}

func (m *e2eProjectManager) CreateProject(_ context.Context, name string, _ ...domain.AddProjectOpts) (*domain.AddProjectResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if _, ok := m.store.FindByRepoName(name); ok {
		return nil, domain.ErrProjectExists
	}

	p := domain.Project{
		Slug:         name,
		RepoName:     name,
		Active:       true,
		RegisteredAt: time.Now(),
	}
	if err := m.store.Add(p); err != nil {
		return nil, err
	}
	return &domain.AddProjectResult{Project: p}, nil
}

func (m *e2eProjectManager) RemoveProject(_ context.Context, slug string, _ bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	err := m.store.Remove(slug)
	if err != nil {
		// Map store errors to domain errors for proper API status codes.
		if strings.Contains(err.Error(), "not found") {
			return domain.ErrProjectNotFound
		}
		return err
	}
	return nil
}

func (m *e2eProjectManager) SyncMirror(_ context.Context, _ string) error {
	return nil
}

// e2eTrackReader is a no-op TrackReader for E2E tests.
type e2eTrackReader struct{}

func (e2eTrackReader) DiscoverTracks(_ string) ([]port.TrackEntry, error) { return nil, nil }
func (e2eTrackReader) DiscoverTracksPaginated(_ string, _ domain.PageOpts, _ ...string) (domain.Page[port.TrackEntry], error) {
	return domain.Page[port.TrackEntry]{}, nil
}
func (e2eTrackReader) GetTrackDetail(_, _ string) (*port.TrackDetail, error) { return nil, nil }
func (e2eTrackReader) RemoveTrack(_, _ string) error                         { return nil }
func (e2eTrackReader) IsInitialized(_ string) bool                           { return false }

// deriveSlug extracts a project name from a git remote URL.
func deriveSlug(url string) string {
	// Strip trailing .git
	url = strings.TrimSuffix(url, ".git")
	// Take last path component
	if idx := strings.LastIndex(url, "/"); idx >= 0 {
		return url[idx+1:]
	}
	if idx := strings.LastIndex(url, ":"); idx >= 0 {
		return url[idx+1:]
	}
	return ""
}
