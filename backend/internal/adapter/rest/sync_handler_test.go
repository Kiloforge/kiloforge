package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
)

// stubProjectStore implements ProjectLister with a fixed set of projects.
type stubProjectStore struct {
	projects []domain.Project
}

func (s *stubProjectStore) List() []domain.Project { return s.projects }

// setupSyncTestMux creates a mux with the sync endpoints wired to real git repos.
func setupSyncTestMux(t *testing.T, projects []domain.Project) *http.ServeMux {
	t.Helper()

	lockMgr := lock.New(t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	lockMgr.StartReaper(ctx)

	store := &stubAgentLister{}
	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:   store,
		LockMgr:  lockMgr,
		Projects: &stubProjectStore{projects: projects},
		GitSync:  gitadapter.New(),
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	mux := http.NewServeMux()
	gen.HandlerFromMux(strictHandler, mux)
	return mux
}

// initTestRepo creates a bare + clone pair and returns a Project pointing at the clone.
func initTestRepo(t *testing.T, slug string) (domain.Project, string) {
	t.Helper()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}

	run("git", "init", "--bare", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	exec.Command("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	exec.Command("git", "-C", tmpWork, "add", ".").Run()
	exec.Command("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	exec.Command("git", "-C", tmpWork, "push", "origin", "main").Run()

	run("git", "clone", bareDir, cloneDir)
	exec.Command("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", cloneDir, "config", "user.name", "Test").Run()

	p := domain.Project{
		Slug:         slug,
		RepoName:     slug,
		ProjectDir:   cloneDir,
		OriginRemote: bareDir,
		Active:       true,
	}
	return p, bareDir
}

func TestGetSyncStatus_OK(t *testing.T) {
	t.Parallel()
	p, _ := initTestRepo(t, "myapp")
	mux := setupSyncTestMux(t, []domain.Project{p})

	req := httptest.NewRequest("GET", "/api/projects/myapp/sync-status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp gen.SyncStatusResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Status != gen.Synced {
		t.Errorf("status = %q, want %q", resp.Status, gen.Synced)
	}
	if resp.LocalBranch != "main" {
		t.Errorf("local_branch = %q, want %q", resp.LocalBranch, "main")
	}
}

func TestGetSyncStatus_NotFound(t *testing.T) {
	t.Parallel()
	mux := setupSyncTestMux(t, nil)

	req := httptest.NewRequest("GET", "/api/projects/nonexistent/sync-status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestPushProject_OK(t *testing.T) {
	t.Parallel()
	p, bareDir := initTestRepo(t, "myapp")

	// Make a local commit to push.
	f, _ := os.Create(filepath.Join(p.ProjectDir, "new.txt"))
	f.WriteString("push me")
	f.Close()
	exec.Command("git", "-C", p.ProjectDir, "add", ".").Run()
	exec.Command("git", "-C", p.ProjectDir, "commit", "-m", "to push").Run()

	mux := setupSyncTestMux(t, []domain.Project{p})

	body, _ := json.Marshal(gen.PushProjectRequest{RemoteBranch: "kf/main"})
	req := httptest.NewRequest("POST", "/api/projects/myapp/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp gen.PushResult
	json.NewDecoder(rec.Body).Decode(&resp)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.RemoteBranch != "kf/main" {
		t.Errorf("remote_branch = %q, want %q", resp.RemoteBranch, "kf/main")
	}

	// Verify remote branch exists.
	out, _ := exec.Command("git", "-C", bareDir, "branch", "--list", "kf/main").Output()
	if len(out) == 0 {
		t.Error("remote branch kf/main not found")
	}
}

func TestPushProject_NotFound(t *testing.T) {
	t.Parallel()
	mux := setupSyncTestMux(t, nil)

	body, _ := json.Marshal(gen.PushProjectRequest{RemoteBranch: "kf/main"})
	req := httptest.NewRequest("POST", "/api/projects/nonexistent/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestPushProject_MissingBranch(t *testing.T) {
	t.Parallel()
	p, _ := initTestRepo(t, "myapp")
	mux := setupSyncTestMux(t, []domain.Project{p})

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/api/projects/myapp/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestPullProject_OK(t *testing.T) {
	t.Parallel()
	p, _ := initTestRepo(t, "myapp")

	// Push an upstream change via a second clone.
	dir := filepath.Dir(p.ProjectDir)
	tmpWork := filepath.Join(dir, "tmp-push")
	exec.Command("git", "clone", filepath.Join(dir, "origin.git"), tmpWork).Run()
	exec.Command("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f.WriteString("upstream")
	f.Close()
	exec.Command("git", "-C", tmpWork, "add", ".").Run()
	exec.Command("git", "-C", tmpWork, "commit", "-m", "upstream").Run()
	exec.Command("git", "-C", tmpWork, "push", "origin", "main").Run()

	mux := setupSyncTestMux(t, []domain.Project{p})

	body, _ := json.Marshal(gen.PullProjectRequest{})
	req := httptest.NewRequest("POST", "/api/projects/myapp/pull", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp gen.PullResult
	json.NewDecoder(rec.Body).Decode(&resp)
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.NewHead == "" {
		t.Error("expected non-empty new_head")
	}
}

func TestPullProject_NotFound(t *testing.T) {
	t.Parallel()
	mux := setupSyncTestMux(t, nil)

	body, _ := json.Marshal(gen.PullProjectRequest{})
	req := httptest.NewRequest("POST", "/api/projects/nonexistent/pull", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
