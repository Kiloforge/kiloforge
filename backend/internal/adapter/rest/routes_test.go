package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"crelay/internal/adapter/badge"
	"crelay/internal/adapter/config"
	"crelay/internal/adapter/dashboard"
	"crelay/internal/adapter/lock"
	"crelay/internal/adapter/persistence/jsonfile"
	"crelay/internal/adapter/rest/gen"
	"crelay/internal/core/domain"
)

// buildMux replicates the route registration logic from Server.Run()
// without calling ListenAndServe. This lets us catch route conflicts
// at test time instead of at runtime.
func buildMux(t *testing.T, srv *Server, dash *dashboard.Server) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("/webhook", srv.handleWebhook)

	lockMgr := lock.New(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	lockMgr.StartReaper(ctx)

	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:  srv.store,
		LockMgr: lockMgr,
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	prLoader := func(slug string) (*domain.PRTracking, error) {
		return nil, nil
	}
	badgeHandler := badge.NewHandler(srv.store, prLoader)
	badgeHandler.RegisterRoutes(mux)

	if dash != nil {
		dash.RegisterNonAPIRoutes(mux)
	}

	return mux
}

// TestRouteRegistration verifies all routes register on a ServeMux without
// panicking. This catches the exact class of bug where duplicate or
// conflicting patterns cause a runtime panic on server startup.
func TestRouteRegistration(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dir,
		GiteaAdminUser: "conductor",
	}
	reg := &jsonfile.ProjectStore{
		Version:  1,
		Projects: map[string]domain.Project{},
	}
	srv := NewServer(cfg, reg, 0)

	mux := buildMux(t, srv, nil)

	routes := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/health", http.StatusOK},
		{"GET", "/-/api/agents", http.StatusOK},
		{"GET", "/-/api/status", http.StatusOK},
		{"GET", "/-/api/quota", http.StatusOK},
		{"GET", "/-/api/locks", http.StatusOK},
	}

	for _, tt := range routes {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("%s %s: got %d, want %d", tt.method, tt.path, rec.Code, tt.want)
			}
		})
	}
}

// TestRouteRegistrationWithDashboard verifies route registration with
// dashboard enabled — the configuration most likely to cause conflicts
// due to overlapping route patterns.
func TestRouteRegistrationWithDashboard(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dir,
		GiteaAdminUser: "conductor",
	}
	reg := &jsonfile.ProjectStore{
		Version:  1,
		Projects: map[string]domain.Project{},
	}
	srv := NewServer(cfg, reg, 0)
	dash := dashboard.New(0, &stubAgentLister{}, nil, "http://localhost:3000", dir)

	mux := buildMux(t, srv, dash)

	// Verify a dashboard-specific route works.
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /health: got %d, want 200", rec.Code)
	}
}

// TestRouteRegistrationWithGiteaProxy verifies that all routes register
// without conflict when the Gitea catch-all proxy is mounted at "/".
// This is the production configuration that previously caused a panic.
func TestRouteRegistrationWithGiteaProxy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dir,
		GiteaAdminUser: "conductor",
	}
	reg := &jsonfile.ProjectStore{
		Version:  1,
		Projects: map[string]domain.Project{},
	}
	srv := NewServer(cfg, reg, 0)
	dash := dashboard.New(0, &stubAgentLister{}, nil, "http://localhost:3000", dir)

	mux := buildMux(t, srv, dash)

	// Mount Gitea proxy as catch-all — same as production Server.Run().
	fakeGitea := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("gitea"))
	}))
	t.Cleanup(fakeGitea.Close)

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("gitea-proxy"))
	}))

	// crelay routes must still work — proxy is only for unmatched paths.
	crelayRoutes := []struct {
		path string
		want int
	}{
		{"/health", http.StatusOK},
		{"/-/api/agents", http.StatusOK},
		{"/-/api/locks", http.StatusOK},
		{"/-/api/badges/track/test", http.StatusOK},
	}
	for _, tt := range crelayRoutes {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != tt.want {
			t.Errorf("GET %s: got %d, want %d", tt.path, rec.Code, tt.want)
		}
	}

	// Gitea asset paths should fall through to the proxy.
	giteaPaths := []string{"/assets/css/theme.css", "/user/login"}
	for _, path := range giteaPaths {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Body.String() != "gitea-proxy" {
			t.Errorf("GET %s: expected gitea proxy, got %q", path, rec.Body.String())
		}
	}
}

// TestBadgeRoutes verifies badge routes register and return SVG content.
func TestBadgeRoutes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServerWithDir(dir)
	mux := buildMux(t, srv, nil)

	req := httptest.NewRequest("GET", "/-/api/badges/track/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("badge route: got %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "image/svg+xml" {
		t.Errorf("badge content-type: got %q, want %q", ct, "image/svg+xml")
	}
}
