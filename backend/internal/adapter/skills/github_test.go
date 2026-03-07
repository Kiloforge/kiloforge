package skills

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestRelease_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name": "v1.2.0",
			"tarball_url": "https://api.github.com/repos/owner/repo/tarball/v1.2.0",
			"published_at": "2026-03-01T00:00:00Z"
		}`))
	}))
	defer srv.Close()

	client := NewGitHubClientWith(srv.Client())
	// Override URL by using the test server
	rel, err := fetchRelease(client.httpClient, srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel.TagName != "v1.2.0" {
		t.Errorf("tag = %q, want v1.2.0", rel.TagName)
	}
	if rel.TarballURL != "https://api.github.com/repos/owner/repo/tarball/v1.2.0" {
		t.Errorf("tarball_url = %q", rel.TarballURL)
	}
}

func TestLatestRelease_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewGitHubClientWith(srv.Client())
	_, err := fetchRelease(client.httpClient, srv.URL)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestLatestRelease_RateLimited(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := NewGitHubClientWith(srv.Client())
	_, err := fetchRelease(client.httpClient, srv.URL)
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestLatestRelease_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := NewGitHubClientWith(srv.Client())
	_, err := fetchRelease(client.httpClient, srv.URL)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
