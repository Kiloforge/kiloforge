package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewGiteaProxy_ForwardsRequest(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("gitea response"))
	}))
	defer backend.Close()

	proxy := NewGiteaProxy(backend.URL)

	// Simulate request to /gitea/some/path — prefix should be stripped.
	mux := http.NewServeMux()
	mux.Handle("/gitea/", http.StripPrefix("/gitea", proxy))

	req := httptest.NewRequest("GET", "/gitea/some/path", nil)
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

	proxy := NewGiteaProxy(backend.URL)

	mux := http.NewServeMux()
	mux.Handle("/gitea/", http.StripPrefix("/gitea", proxy))

	req := httptest.NewRequest("GET", "/gitea/", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if w.Header().Get("X-Backend-Path") != "/" {
		t.Errorf("backend path = %q, want /", w.Header().Get("X-Backend-Path"))
	}
}
