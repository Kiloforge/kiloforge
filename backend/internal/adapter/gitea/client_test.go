package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientWithToken_UsesTokenAuth(t *testing.T) {
	t.Parallel()

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version": "1.0"}`))
	}))
	defer srv.Close()

	client := NewClientWithToken(srv.URL, "admin", "mytoken123")
	_, err := client.CheckVersion(context.Background())
	if err != nil {
		t.Fatalf("CheckVersion: %v", err)
	}
	if gotAuth != "token mytoken123" {
		t.Errorf("Authorization header: want %q, got %q", "token mytoken123", gotAuth)
	}
}

func TestNewClientWithToken_NoBasicAuth(t *testing.T) {
	t.Parallel()

	var gotBasicUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _, ok := r.BasicAuth()
		if ok {
			gotBasicUser = user
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version": "1.0"}`))
	}))
	defer srv.Close()

	client := NewClientWithToken(srv.URL, "admin", "mytoken123")
	_, _ = client.CheckVersion(context.Background())
	if gotBasicUser != "" {
		t.Errorf("BasicAuth should not be set when token is used, got user %q", gotBasicUser)
	}
}

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

	client := NewClient(srv.URL, "kiloforger", "pass")
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

	client := NewClient(srv.URL, "kiloforger", "pass")
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

	client := NewClient(srv.URL, "kiloforger", "pass")
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

	client := NewClient(srv.URL, "kiloforger", "pass")
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

	client := NewClient(srv.URL, "kiloforger", "pass")
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

func TestDeleteRepo_Success(t *testing.T) {
	t.Parallel()

	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	err := client.DeleteRepo(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("DeleteRepo: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method: want DELETE, got %s", gotMethod)
	}
	if gotPath != "/api/v1/repos/conductor/myapp" {
		t.Errorf("path: want /api/v1/repos/conductor/myapp, got %s", gotPath)
	}
}

func TestDeleteRepo_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "repo not found"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	err := client.DeleteRepo(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestListWebhooks(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/conductor/myapp/hooks" || r.Method != "GET" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	hooks, err := client.ListWebhooks(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("ListWebhooks: %v", err)
	}
	if len(hooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(hooks))
	}
}

func TestDeleteAllWebhooks(t *testing.T) {
	t.Parallel()

	deletedIDs := make(map[string]bool)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v1/repos/conductor/myapp/hooks":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": 10}, {"id": 20}]`))
		case r.Method == "DELETE":
			deletedIDs[r.URL.Path] = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "kiloforger", "pass")
	err := client.DeleteAllWebhooks(context.Background(), "myapp")
	if err != nil {
		t.Fatalf("DeleteAllWebhooks: %v", err)
	}
	if len(deletedIDs) != 2 {
		t.Errorf("expected 2 delete calls, got %d", len(deletedIDs))
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

	client := NewClient(srv.URL, "kiloforger", "pass")
	reviews, err := client.GetPRReviews(context.Background(), "myapp", 5)
	if err != nil {
		t.Fatalf("GetPRReviews: %v", err)
	}
	if len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(reviews))
	}
}
