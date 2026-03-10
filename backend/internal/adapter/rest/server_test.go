package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/adapter/skills"
	"kiloforge/internal/core/domain"
)

func newTestServer() *Server {
	return newTestServerWithDir("")
}

func newTestServerWithDir(dataDir string) *Server {
	if dataDir == "" {
		var err error
		dataDir, err = os.MkdirTemp("", "kf-test-data-*")
		if err != nil {
			panic(fmt.Sprintf("create test data dir: %v", err))
		}
	}

	// Install embedded skills for validation to pass.
	skillsDir, _ := os.MkdirTemp("", "kf-test-skills-*")
	for _, name := range skills.ListEmbedded() {
		skills.InstallEmbedded(name, skillsDir)
	}

	cfg := &config.Config{
		DataDir:   dataDir,
		SkillsDir: skillsDir,
	}
	db, err := sqlite.Open(dataDir)
	if err != nil {
		panic(fmt.Sprintf("open test db: %v", err))
	}
	reg := sqlite.NewProjectStore(db)
	if err := reg.Add(domain.Project{Slug: "myapp", RepoName: "myapp"}); err != nil {
		panic(fmt.Sprintf("seed project: %v", err))
	}
	store := sqlite.NewAgentStore(db)
	prTracker := sqlite.NewPRTrackingStore(db)
	return NewServer(cfg, reg, store, prTracker, 3001)
}

func TestHealth_ReportsProjectCount(t *testing.T) {
	t.Parallel()

	h := NewAPIHandler(APIHandlerOpts{
		Agents:   &stubAgentLister{},
		LockMgr:  lock.New(""),
		Projects: &stubProjectLister{projects: []domain.Project{{Slug: "test"}}},
	})
	strictHandler := gen.NewStrictHandler(h, nil)
	mux := http.NewServeMux()
	gen.HandlerFromMux(strictHandler, mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
	if int(resp["projects"].(float64)) != 1 {
		t.Errorf("expected 1 project, got %v", resp["projects"])
	}
}

func TestNewServer_WithDashboard(t *testing.T) {
	t.Parallel()
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
	srv := NewServer(cfg, reg, store, sqlite.NewPRTrackingStore(db), 3001, WithDashboard(nil, nil, "/", &stubProjectLister{}))

	if srv.dashboard == nil {
		t.Fatal("expected dashboard to be set")
	}
}

func TestNewServer_WithoutDashboard(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	if srv.dashboard != nil {
		t.Fatal("expected dashboard to be nil")
	}
}
