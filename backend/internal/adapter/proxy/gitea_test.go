package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewGiteaProxy_InjectsAuthHeader(t *testing.T) {
	t.Parallel()

	var gotHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-WEBAUTH-USER")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := NewGiteaProxy(backend.URL, "kfadmin")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, req)

	if gotHeader != "kfadmin" {
		t.Errorf("X-WEBAUTH-USER = %q, want %q", gotHeader, "kfadmin")
	}
}

func TestNewGiteaProxy_NoAuthUser(t *testing.T) {
	t.Parallel()

	var gotHeader string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-WEBAUTH-USER")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := NewGiteaProxy(backend.URL, "")

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	proxy.ServeHTTP(w, req)

	if gotHeader != "" {
		t.Errorf("X-WEBAUTH-USER should be empty when no authUser, got %q", gotHeader)
	}
}

func TestNewGiteaProxy_ForwardsRequest(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("gitea response"))
	}))
	defer backend.Close()

	proxy := NewGiteaProxy(backend.URL, "admin")

	// Proxy is mounted as catch-all at "/" — no prefix stripping.
	mux := http.NewServeMux()
	mux.Handle("/", proxy)

	req := httptest.NewRequest("GET", "/some/path", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	body, _ := io.ReadAll(w.Body)
	if string(body) != "gitea response" {
		t.Errorf("body = %q, want %q", body, "gitea response")
	}
	if w.Header().Get("X-Backend-Path") != "/some/path" {
		t.Errorf("backend path = %q, want /some/path", w.Header().Get("X-Backend-Path"))
	}
}

func TestNewGiteaProxy_RootPath(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := NewGiteaProxy(backend.URL, "admin")

	mux := http.NewServeMux()
	mux.Handle("/", proxy)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Header().Get("X-Backend-Path") != "/" {
		t.Errorf("backend path = %q, want /", w.Header().Get("X-Backend-Path"))
	}
}

func TestNewGiteaProxy_AssetPaths(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	proxy := NewGiteaProxy(backend.URL, "admin")

	mux := http.NewServeMux()
	mux.Handle("/", proxy)

	// Gitea asset paths must reach the backend without any prefix mangling.
	paths := []string{
		"/assets/css/theme-gitea-auto.css",
		"/assets/js/index.js",
		"/user/login",
		"/api/v1/version",
	}

	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", path, w.Code)
		}
		if got := w.Header().Get("X-Backend-Path"); got != path {
			t.Errorf("GET %s: backend saw %q, want %q", path, got, path)
		}
	}
}
