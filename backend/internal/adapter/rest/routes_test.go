package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"kiloforge/internal/adapter/badge"
	"kiloforge/internal/adapter/config"
	"kiloforge/internal/adapter/dashboard"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
)

// buildMux replicates the route registration logic from Server.Run()
// without calling ListenAndServe. This lets us catch route conflicts
// at test time instead of at runtime.
func buildMux(t *testing.T, srv *Server, dash *dashboard.Server) *http.ServeMux {
	t.Helper()
	mux := http.NewServeMux()

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
		DataDir: dir,
	}
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	reg := sqlite.NewProjectStore(db)
	store := sqlite.NewAgentStore(db)
	srv := NewServer(cfg, reg, store, sqlite.NewPRTrackingStore(db), 0)

	mux := buildMux(t, srv, nil)

	routes := []struct {
		method string
		path   string
		want   int
	}{
		{"GET", "/health", http.StatusOK},
		{"GET", "/api/agents", http.StatusOK},
		{"GET", "/api/status", http.StatusOK},
		{"GET", "/api/quota", http.StatusOK},
		{"GET", "/api/locks", http.StatusOK},
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
		DataDir: dir,
	}
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	reg := sqlite.NewProjectStore(db)
	store := sqlite.NewAgentStore(db)
	srv := NewServer(cfg, reg, store, sqlite.NewPRTrackingStore(db), 0)
	dash := dashboard.New(0, &stubAgentLister{}, nil, &stubProjectLister{}, nil)

	mux := buildMux(t, srv, dash)

	// Verify a dashboard-specific route works.
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /health: got %d, want 200", rec.Code)
	}
}

// TestBadgeRoutes verifies badge routes register and return SVG content.
func TestBadgeRoutes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServerWithDir(dir)
	mux := buildMux(t, srv, nil)

	req := httptest.NewRequest("GET", "/api/badges/track/nonexistent", nil)
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
