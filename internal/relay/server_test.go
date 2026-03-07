package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"crelay/internal/config"
	"crelay/internal/orchestration"
	"crelay/internal/project"
	"crelay/internal/state"
)

func newTestServer() *Server {
	return newTestServerWithDir("")
}

func newTestServerWithDir(dataDir string) *Server {
	if dataDir == "" {
		dataDir = "/tmp/crelay-test-" + fmt.Sprintf("%d", os.Getpid())
	}
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dataDir,
		GiteaAdminUser: "conductor",
	}
	reg := &project.Registry{
		Version: 1,
		Projects: map[string]project.Project{
			"myapp": {
				Slug:     "myapp",
				RepoName: "myapp",
			},
		},
	}
	return NewServer(cfg, reg, 3001)
}

func postWebhook(t *testing.T, srv *Server, event string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Gitea-Event", event)
	rec := httptest.NewRecorder()
	srv.handleWebhook(rec, req)
	return rec
}

func TestHandleWebhook_UnknownRepo(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	rec := postWebhook(t, srv, "push", map[string]any{
		"repository": map[string]any{"name": "unknown-repo"},
		"ref":        "refs/heads/main",
		"commits":    []any{},
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ignored" {
		t.Errorf("expected status 'ignored', got %q", resp["status"])
	}
}

func TestHandleWebhook_IssueOpened(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	rec := postWebhook(t, srv, "issues", map[string]any{
		"action": "opened",
		"repository": map[string]any{"name": "myapp"},
		"issue": map[string]any{
			"number": float64(7),
			"title":  "Fix login bug",
		},
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleWebhook_IssueClosed(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	rec := postWebhook(t, srv, "issues", map[string]any{
		"action": "closed",
		"repository": map[string]any{"name": "myapp"},
		"issue": map[string]any{
			"number": float64(7),
			"title":  "Fix login bug",
		},
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleWebhook_IssueComment(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	rec := postWebhook(t, srv, "issue_comment", map[string]any{
		"action": "created",
		"repository": map[string]any{"name": "myapp"},
		"issue": map[string]any{
			"number": float64(7),
		},
		"comment": map[string]any{
			"body": "This needs tests",
		},
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleWebhook_PullRequest(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	rec := postWebhook(t, srv, "pull_request", map[string]any{
		"action": "opened",
		"repository": map[string]any{"name": "myapp"},
		"pull_request": map[string]any{
			"number": float64(3),
			"title":  "Add auth module",
		},
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleWebhook_PullRequestReview(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	rec := postWebhook(t, srv, "pull_request_review", map[string]any{
		"action": "submitted",
		"repository": map[string]any{"name": "myapp"},
		"review": map[string]any{
			"state": "approved",
		},
		"pull_request": map[string]any{
			"number": float64(3),
		},
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleWebhook_Push(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	rec := postWebhook(t, srv, "push", map[string]any{
		"repository": map[string]any{"name": "myapp"},
		"ref":        "refs/heads/main",
		"commits":    []any{map[string]any{"id": "abc123"}},
	})

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHealth_ReportsProjectCount(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.handleHealth(rec, req)

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", resp["status"])
	}
	if int(resp["projects"].(float64)) != 1 {
		t.Errorf("expected 1 project, got %v", resp["projects"])
	}
}

func TestResolveProject(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	proj, ok := srv.resolveProject(map[string]any{
		"repository": map[string]any{"name": "myapp"},
	})
	if !ok {
		t.Fatal("expected to resolve project")
	}
	if proj.Slug != "myapp" {
		t.Errorf("expected slug 'myapp', got %q", proj.Slug)
	}

	_, ok = srv.resolveProject(map[string]any{
		"repository": map[string]any{"name": "unknown"},
	})
	if ok {
		t.Error("expected not to resolve unknown project")
	}

	_, ok = srv.resolveProject(map[string]any{})
	if ok {
		t.Error("expected not to resolve without repository")
	}
}

func TestHandleWebhook_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	srv := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	rec := httptest.NewRecorder()
	srv.handleWebhook(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestHandleWebhook_PROpened_CreatesTracking(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srv := newTestServerWithDir(dir)

	// Add a developer agent to state so the tracking can find it.
	srv.store.AddAgent(state.AgentInfo{
		ID:        "dev-agent-123",
		Role:      "developer",
		Ref:       "my-track_20260101Z",
		Status:    "running",
		SessionID: "dev-session-456",
	})

	rec := postWebhook(t, srv, "pull_request", map[string]any{
		"action":     "opened",
		"repository": map[string]any{"name": "myapp"},
		"pull_request": map[string]any{
			"number": float64(5),
			"title":  "feat: my track",
			"head": map[string]any{
				"ref": "my-track_20260101Z",
			},
		},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Verify tracking record created.
	projectDir := filepath.Join(dir, "projects", "myapp")
	tracking, err := orchestration.LoadPRTracking(projectDir)
	if err != nil {
		t.Fatalf("LoadPRTracking: %v", err)
	}
	if tracking.PRNumber != 5 {
		t.Errorf("PRNumber: want 5, got %d", tracking.PRNumber)
	}
	if tracking.TrackID != "my-track_20260101Z" {
		t.Errorf("TrackID: want %q, got %q", "my-track_20260101Z", tracking.TrackID)
	}
	if tracking.DeveloperAgentID != "dev-agent-123" {
		t.Errorf("DeveloperAgentID: want %q, got %q", "dev-agent-123", tracking.DeveloperAgentID)
	}
	if tracking.Status != "waiting-review" {
		t.Errorf("Status: want %q, got %q", "waiting-review", tracking.Status)
	}
}
