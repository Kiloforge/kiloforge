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

func TestCommentOnPR(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	err := client.CommentOnPR(context.Background(), "myapp", 5, "Review cycle limit reached.")
	if err != nil {
		t.Fatalf("CommentOnPR: %v", err)
	}
	if gotPath != "/api/v1/repos/conductor/myapp/issues/5/comments" {
		t.Errorf("path: want /api/v1/repos/conductor/myapp/issues/5/comments, got %s", gotPath)
	}
	if gotPayload["body"] != "Review cycle limit reached." {
		t.Errorf("body: want %q, got %v", "Review cycle limit reached.", gotPayload["body"])
	}
}

func TestAddLabel(t *testing.T) {
	t.Parallel()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch {
		case r.Method == "POST" && r.URL.Path == "/api/v1/repos/conductor/myapp/labels":
			// Create label
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 42, "name": "needs-human-review"}`))
		case r.Method == "POST" && r.URL.Path == "/api/v1/repos/conductor/myapp/issues/5/labels":
			// Add label to issue
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": 42}]`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	err := client.AddLabel(context.Background(), "myapp", 5, "needs-human-review")
	if err != nil {
		t.Fatalf("AddLabel: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 API calls, got %d", calls)
	}
}

func TestMergePR_Success(t *testing.T) {
	t.Parallel()

	var gotPayload map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	err := client.MergePR(context.Background(), "myapp", 3, "merge")
	if err != nil {
		t.Fatalf("MergePR: %v", err)
	}
	if gotPath != "/api/v1/repos/conductor/myapp/pulls/3/merge" {
		t.Errorf("path: want /api/v1/repos/conductor/myapp/pulls/3/merge, got %s", gotPath)
	}
	if gotPayload["Do"] != "merge" {
		t.Errorf("Do: want %q, got %v", "merge", gotPayload["Do"])
	}
}

func TestMergePR_Conflict(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message": "merge conflict"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	err := client.MergePR(context.Background(), "myapp", 3, "merge")
	if err == nil {
		t.Fatal("expected error on merge conflict")
	}
}

func TestDeleteBranch(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	err := client.DeleteBranch(context.Background(), "myapp", "feature-branch")
	if err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method: want DELETE, got %s", gotMethod)
	}
	if gotPath != "/api/v1/repos/conductor/myapp/branches/feature-branch" {
		t.Errorf("path: want /api/v1/repos/conductor/myapp/branches/feature-branch, got %s", gotPath)
	}
}

func TestGetPRReviews(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/conductor/myapp/pulls/5/reviews" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1, "state": "REQUEST_CHANGES", "body": "Fix the tests"}]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "conductor", "pass")
	reviews, err := client.GetPRReviews(context.Background(), "myapp", 5)
	if err != nil {
		t.Fatalf("GetPRReviews: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(reviews))
	}
}
