package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddSSHKey_Success(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v1/user/keys" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "admin", "pass")
	err := client.AddSSHKey(context.Background(), "test-key", "ssh-ed25519 AAAA")
	if err != nil {
		t.Fatalf("AddSSHKey: %v", err)
	}
	if gotPayload["title"] != "test-key" {
		t.Errorf("title: want %q, got %v", "test-key", gotPayload["title"])
	}
	if gotPayload["key"] != "ssh-ed25519 AAAA" {
		t.Errorf("key: want %q, got %v", "ssh-ed25519 AAAA", gotPayload["key"])
	}
}

func TestAddSSHKey_AlreadyExists(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"message": "key already exists"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "admin", "pass")
	err := client.AddSSHKey(context.Background(), "test-key", "ssh-ed25519 AAAA")
	if err != nil {
		t.Fatalf("AddSSHKey should not error on 422, got: %v", err)
	}
}

func TestAddSSHKey_OtherError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "server error"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "admin", "pass")
	err := client.AddSSHKey(context.Background(), "test-key", "ssh-ed25519 AAAA")
	if err == nil {
		t.Fatal("AddSSHKey should return error on 500")
	}
}
