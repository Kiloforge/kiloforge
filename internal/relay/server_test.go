package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"crelay/internal/config"
	"crelay/internal/core/domain"
	"crelay/internal/gitea"
	"crelay/internal/orchestration"
	"crelay/internal/project"
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
		Projects: map[string]domain.Project{
			"myapp": {
				Slug:     "myapp",
				RepoName: "myapp",
			},
		},
	}
	return NewServer(cfg, reg, 3001)
}

// newTestServerWithSpawner creates a server with a fake spawner for testing review cycle.
func newTestServerWithSpawner(dataDir string, spawner AgentSpawner, giteaSrv *httptest.Server) *Server {
	cfg := &config.Config{
		GiteaPort:      3000,
		DataDir:        dataDir,
		GiteaAdminUser: "conductor",
	}
	reg := &project.Registry{
		Version: 1,
		Projects: map[string]domain.Project{
			"myapp": {
				Slug:     "myapp",
				RepoName: "myapp",
			},
		},
	}
	var client *gitea.Client
	if giteaSrv != nil {
		client = gitea.NewClient(giteaSrv.URL, "conductor", "pass")
	} else {
		client = gitea.NewClient("http://localhost:3000", "conductor", "pass")
	}
	return newTestableServer(cfg, reg, spawner, client)
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
	spawner := &fakeSpawner{}
	srv := newTestServerWithSpawner(dir, spawner, nil)

	// Add a developer agent to state so the tracking can find it.
	srv.store.AddAgent(domain.AgentInfo{
		ID:          "dev-agent-123",
		Role:        "developer",
		Ref:         "my-track_20260101Z",
		Status:      "running",
		SessionID:   "dev-session-456",
		WorktreeDir: "/tmp/worktree",
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

	// Reviewer should have been spawned.
	if len(spawner.reviewerCalls) != 1 {
		t.Fatalf("expected 1 reviewer spawn, got %d", len(spawner.reviewerCalls))
	}
}

func TestReviewApproved_MergesAndCleans(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spawner := &fakeSpawner{}

	// Fake Gitea server to receive merge/comment/delete-branch calls.
	var giteaCalls []string
	giteaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		giteaCalls = append(giteaCalls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer giteaSrv.Close()

	srv := newTestServerWithSpawner(dir, spawner, giteaSrv)

	// Set up developer agent.
	srv.store.AddAgent(domain.AgentInfo{
		ID:        "dev-agent-123",
		Role:      "developer",
		Ref:       "my-track",
		Status:    "waiting-review",
		SessionID: "dev-session-456",
	})

	// Create PR tracking.
	projectDir := filepath.Join(dir, "projects", "myapp")
	os.MkdirAll(projectDir, 0o755)
	tracking := &domain.PRTracking{
		PRNumber:         5,
		TrackID:          "my-track",
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-agent-123",
		DeveloperSession: "dev-session-456",
		MaxReviewCycles:  3,
		Status:           "in-review",
	}
	orchestration.SavePRTracking(tracking, projectDir)

	// Send approved review.
	rec := postWebhook(t, srv, "pull_request_review", map[string]any{
		"action":     "submitted",
		"repository": map[string]any{"name": "myapp"},
		"review":     map[string]any{"state": "approved"},
		"pull_request": map[string]any{
			"number": float64(5),
		},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Should NOT have resumed developer (merge+cleanup instead).
	if len(spawner.resumeCalls) != 0 {
		t.Errorf("expected 0 resume calls, got %d", len(spawner.resumeCalls))
	}

	// Should have called merge API.
	hasMerge := false
	for _, call := range giteaCalls {
		if call == "POST /api/v1/repos/conductor/myapp/pulls/5/merge" {
			hasMerge = true
		}
	}
	if !hasMerge {
		t.Errorf("expected merge API call, got: %v", giteaCalls)
	}

	// Tracking should be merged.
	updated, _ := orchestration.LoadPRTracking(projectDir)
	if updated.Status != "merged" {
		t.Errorf("status: want %q, got %q", "merged", updated.Status)
	}

	// Developer agent should be completed.
	dev, _ := srv.store.FindAgent("dev-agent-123")
	if dev.Status != "completed" {
		t.Errorf("developer status: want %q, got %q", "completed", dev.Status)
	}
}

func TestReviewChangesRequested_ResumesDeveloper(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spawner := &fakeSpawner{}
	srv := newTestServerWithSpawner(dir, spawner, nil)

	srv.store.AddAgent(domain.AgentInfo{
		ID:        "dev-agent-123",
		Role:      "developer",
		Ref:       "my-track",
		Status:    "waiting-review",
		SessionID: "dev-session-456",
	})

	projectDir := filepath.Join(dir, "projects", "myapp")
	os.MkdirAll(projectDir, 0o755)
	tracking := &domain.PRTracking{
		PRNumber:         5,
		TrackID:          "my-track",
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-agent-123",
		DeveloperSession: "dev-session-456",
		MaxReviewCycles:  3,
		Status:           "in-review",
	}
	orchestration.SavePRTracking(tracking, projectDir)

	rec := postWebhook(t, srv, "pull_request_review", map[string]any{
		"action":     "submitted",
		"repository": map[string]any{"name": "myapp"},
		"review":     map[string]any{"state": "changes_requested"},
		"pull_request": map[string]any{
			"number": float64(5),
		},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if len(spawner.resumeCalls) != 1 {
		t.Fatalf("expected 1 resume, got %d", len(spawner.resumeCalls))
	}

	updated, _ := orchestration.LoadPRTracking(projectDir)
	if updated.ReviewCycleCount != 1 {
		t.Errorf("ReviewCycleCount: want 1, got %d", updated.ReviewCycleCount)
	}
	if updated.Status != "changes-requested" {
		t.Errorf("status: want %q, got %q", "changes-requested", updated.Status)
	}
}

func TestReviewCycleLimit_Escalates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spawner := &fakeSpawner{}

	// Fake Gitea server to receive label/comment calls.
	var giteaCalls []string
	giteaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		giteaCalls = append(giteaCalls, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1}`))
	}))
	defer giteaSrv.Close()

	srv := newTestServerWithSpawner(dir, spawner, giteaSrv)

	srv.store.AddAgent(domain.AgentInfo{
		ID:        "dev-agent-123",
		Role:      "developer",
		Ref:       "my-track",
		Status:    "waiting-review",
		SessionID: "dev-session-456",
	})

	projectDir := filepath.Join(dir, "projects", "myapp")
	os.MkdirAll(projectDir, 0o755)
	tracking := &domain.PRTracking{
		PRNumber:         5,
		TrackID:          "my-track",
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-agent-123",
		DeveloperSession: "dev-session-456",
		ReviewCycleCount: 2, // Already at 2, next will be 3 = limit
		MaxReviewCycles:  3,
		Status:           "in-review",
	}
	orchestration.SavePRTracking(tracking, projectDir)

	postWebhook(t, srv, "pull_request_review", map[string]any{
		"action":     "submitted",
		"repository": map[string]any{"name": "myapp"},
		"review":     map[string]any{"state": "changes_requested"},
		"pull_request": map[string]any{
			"number": float64(5),
		},
	})

	// Should NOT have resumed developer (escalated instead).
	if len(spawner.resumeCalls) != 0 {
		t.Errorf("expected 0 resume calls (escalated), got %d", len(spawner.resumeCalls))
	}

	// Should have made Gitea API calls for label + comment.
	if len(giteaCalls) < 2 {
		t.Errorf("expected at least 2 Gitea API calls, got %d: %v", len(giteaCalls), giteaCalls)
	}

	// Tracking should be escalated.
	updated, _ := orchestration.LoadPRTracking(projectDir)
	if updated.Status != "escalated" {
		t.Errorf("status: want %q, got %q", "escalated", updated.Status)
	}
}

func TestPRSynchronize_SpawnsReviewer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	spawner := &fakeSpawner{}
	srv := newTestServerWithSpawner(dir, spawner, nil)

	srv.store.AddAgent(domain.AgentInfo{
		ID:        "dev-agent-123",
		Role:      "developer",
		Ref:       "my-track",
		Status:    "running",
		SessionID: "dev-session-456",
	})

	projectDir := filepath.Join(dir, "projects", "myapp")
	os.MkdirAll(projectDir, 0o755)
	tracking := &domain.PRTracking{
		PRNumber:         5,
		TrackID:          "my-track",
		ProjectSlug:      "myapp",
		DeveloperAgentID: "dev-agent-123",
		DeveloperSession: "dev-session-456",
		MaxReviewCycles:  3,
		Status:           "changes-requested",
	}
	orchestration.SavePRTracking(tracking, projectDir)

	postWebhook(t, srv, "pull_request", map[string]any{
		"action":     "synchronize",
		"repository": map[string]any{"name": "myapp"},
		"pull_request": map[string]any{
			"number": float64(5),
			"title":  "feat: my track",
		},
	})

	// Reviewer should have been spawned.
	if len(spawner.reviewerCalls) != 1 {
		t.Fatalf("expected 1 reviewer spawn, got %d", len(spawner.reviewerCalls))
	}
}

// fakeSpawner records calls for testing.
type fakeSpawner struct {
	reviewerCalls []ReviewerOpts
	resumeCalls   []resumeCall
}

type resumeCall struct {
	sessionID string
	workDir   string
}

func (f *fakeSpawner) SpawnReviewer(_ context.Context, opts ReviewerOpts) (*domain.AgentInfo, error) {
	f.reviewerCalls = append(f.reviewerCalls, opts)
	return &domain.AgentInfo{
		ID:        "reviewer-fake",
		Role:      "reviewer",
		Ref:       fmt.Sprintf("PR #%d", opts.PRNumber),
		Status:    "running",
		SessionID: "reviewer-session-fake",
	}, nil
}

func (f *fakeSpawner) ResumeDeveloper(_ context.Context, sessionID, workDir string) error {
	f.resumeCalls = append(f.resumeCalls, resumeCall{sessionID: sessionID, workDir: workDir})
	return nil
}
