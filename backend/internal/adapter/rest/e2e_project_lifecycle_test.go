//go:build e2e

package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitadapter "kiloforge/internal/adapter/git"
	"kiloforge/internal/adapter/lock"
	"kiloforge/internal/adapter/persistence/sqlite"
	"kiloforge/internal/adapter/rest/gen"
	"kiloforge/internal/core/domain"
)

// ---------------------------------------------------------------------------
// Phase 1 — Helpers
// ---------------------------------------------------------------------------

// startE2EServerWithGitSync is like startE2EServer but wires gitadapter.New()
// as GitSync so push/pull/sync-status endpoints work with real git repos.
func startE2EServerWithGitSync(t *testing.T) *e2eServer {
	t.Helper()

	mockBin := buildMockAgentBinary(t)

	dir := t.TempDir()
	db, err := sqlite.Open(dir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	reg := sqlite.NewProjectStore(db)
	store := sqlite.NewAgentStore(db)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	ctx, cancel := context.WithCancel(context.Background())

	mux := http.NewServeMux()

	lockMgr := lock.New(dir)
	lockMgr.StartReaper(ctx)

	projectMgr := newE2EProjectManager(reg)

	apiHandler := NewAPIHandler(APIHandlerOpts{
		Agents:     store,
		LockMgr:    lockMgr,
		Projects:   reg,
		ProjectMgr: projectMgr,
		GitSync:    gitadapter.New(),
	})
	strictHandler := gen.NewStrictHandler(apiHandler, nil)
	gen.HandlerFromMux(strictHandler, mux)

	httpSrv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		httpSrv.Shutdown(context.Background())
	}()

	go func() {
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			t.Logf("server error: %v", err)
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(url + "/health"); err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Cleanup(cancel)

	return &e2eServer{
		URL:          url,
		MockAgentBin: mockBin,
		DataDir:      dir,
		cancel:       cancel,
		db:           db,
		projects:     reg,
		agents:       store,
	}
}

// e2eCleanGitCmd creates a git exec.Cmd with GIT_DIR/GIT_WORK_TREE removed
// to prevent worktree env vars from leaking into subprocess git operations.
func e2eCleanGitCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env
	return cmd
}

// initE2ETestRepo creates a bare + clone pair with an initial commit.
// Returns (bareDir, cloneDir).
func initE2ETestRepo(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		var env []string
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") {
				continue
			}
			env = append(env, e)
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s: %v", args, out, err)
		}
	}

	run("git", "init", "--bare", bareDir)

	// Create a temporary working tree to make the initial commit.
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)

	gitCfg := func(args ...string) {
		cmd := e2eCleanGitCmd(args...)
		cmd.Dir = dir
		cmd.Run()
	}
	gitCfg("git", "-C", tmpWork, "config", "user.email", "test@test.com")
	gitCfg("git", "-C", tmpWork, "config", "user.name", "Test")

	if err := os.WriteFile(filepath.Join(tmpWork, "README.md"), []byte("# test"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	run("git", "-C", tmpWork, "add", ".")
	run("git", "-C", tmpWork, "commit", "-m", "initial")
	run("git", "-C", tmpWork, "push", "origin", "main")

	// Create the clone that will act as the managed project directory.
	run("git", "clone", bareDir, cloneDir)
	gitCfg("git", "-C", cloneDir, "config", "user.email", "test@test.com")
	gitCfg("git", "-C", cloneDir, "config", "user.name", "Test")

	return bareDir, cloneDir
}

// ---------------------------------------------------------------------------
// Phase 2 — Add project E2E tests
// ---------------------------------------------------------------------------

func TestE2E_AddLocalProject(t *testing.T) {
	srv := startE2EServerWithGitSync(t)

	// Create a real git repo.
	_, cloneDir := initE2ETestRepo(t)

	body := map[string]string{"local_path": cloneDir}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var project map[string]any
	json.NewDecoder(resp.Body).Decode(&project)
	if project["slug"] != "clone" {
		t.Errorf("expected slug=clone, got %v", project["slug"])
	}
	if project["active"] != true {
		t.Errorf("expected active=true, got %v", project["active"])
	}

	// Verify project appears in list.
	listResp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer listResp.Body.Close()

	var projects []map[string]any
	json.NewDecoder(listResp.Body).Decode(&projects)
	found := false
	for _, p := range projects {
		if p["slug"] == "clone" {
			found = true
			break
		}
	}
	if !found {
		t.Error("project 'clone' not found in project list")
	}
}

func TestE2E_AddRemoteMirror(t *testing.T) {
	srv := startE2EServerWithGitSync(t)

	// Create a bare repo to act as the "remote".
	bareDir, _ := initE2ETestRepo(t)

	body := map[string]string{"remote_url": bareDir}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 201, got %d: %v", resp.StatusCode, errBody)
	}

	var project map[string]any
	json.NewDecoder(resp.Body).Decode(&project)
	slug := project["slug"]
	if slug == nil || slug == "" {
		t.Fatal("expected non-empty slug")
	}
}

func TestE2E_AddLocalProject_InvalidPath(t *testing.T) {
	srv := startE2EServerWithGitSync(t)

	body := map[string]string{"local_path": "/nonexistent/path/does/not/exist"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestE2E_AddLocalProject_NotGitRepo(t *testing.T) {
	srv := startE2EServerWithGitSync(t)

	// Create a directory that is not a git repo.
	dir := t.TempDir()

	body := map[string]string{"local_path": dir}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST /api/projects: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Phase 3 — Remove project E2E tests
// ---------------------------------------------------------------------------

func TestE2E_RemoveProject_Lifecycle(t *testing.T) {
	srv := startE2EServerWithGitSync(t)

	// Add a project via remote URL.
	body := map[string]string{"remote_url": "https://github.com/user/lifecycle-remove.git"}
	b, _ := json.Marshal(body)
	resp, err := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: expected 201, got %d", resp.StatusCode)
	}

	// Delete the project.
	req, _ := http.NewRequest("DELETE", srv.URL+"/api/projects/lifecycle-remove", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}

	// Verify project is gone from list.
	listResp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer listResp.Body.Close()
	var projects []map[string]any
	json.NewDecoder(listResp.Body).Decode(&projects)
	for _, p := range projects {
		if p["slug"] == "lifecycle-remove" {
			t.Error("project 'lifecycle-remove' still in list after deletion")
		}
	}
}

func TestE2E_RemoveProject_WithCleanup_Lifecycle(t *testing.T) {
	srv := startE2EServerWithGitSync(t)

	body := map[string]string{"remote_url": "https://github.com/user/lifecycle-cleanup.git"}
	b, _ := json.Marshal(body)
	resp, _ := http.Post(srv.URL+"/api/projects", "application/json", bytes.NewReader(b))
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add: expected 201, got %d", resp.StatusCode)
	}

	req, _ := http.NewRequest("DELETE", srv.URL+"/api/projects/lifecycle-cleanup?cleanup=true", nil)
	delResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE with cleanup: %v", err)
	}
	defer delResp.Body.Close()

	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}

	// Verify project is gone.
	listResp, err := http.Get(srv.URL + "/api/projects")
	if err != nil {
		t.Fatalf("GET /api/projects: %v", err)
	}
	defer listResp.Body.Close()
	var projects []map[string]any
	json.NewDecoder(listResp.Body).Decode(&projects)
	for _, p := range projects {
		if p["slug"] == "lifecycle-cleanup" {
			t.Error("project still in list after delete with cleanup")
		}
	}
}

// ---------------------------------------------------------------------------
// Phase 4 — Sync E2E tests
// ---------------------------------------------------------------------------

// registerRealProject inserts a project with real git paths directly into the
// E2E server's project store so that GitSync endpoints can operate on it.
func registerRealProject(t *testing.T, srv *e2eServer, slug, cloneDir, bareDir string) {
	t.Helper()
	err := srv.projects.Add(domain.Project{
		Slug:          slug,
		RepoName:      slug,
		ProjectDir:    cloneDir,
		OriginRemote:  bareDir,
		PrimaryBranch: "main",
		Active:        true,
		RegisteredAt:  time.Now(),
	})
	if err != nil {
		t.Fatalf("register project %q: %v", slug, err)
	}
}

func TestE2E_SyncPush(t *testing.T) {
	srv := startE2EServerWithGitSync(t)
	bareDir, cloneDir := initE2ETestRepo(t)
	registerRealProject(t, srv, "sync-push", cloneDir, bareDir)

	// Make a local commit in the clone.
	if err := os.WriteFile(filepath.Join(cloneDir, "new.txt"), []byte("push me"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	e2eCleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	e2eCleanGitCmd("git", "-C", cloneDir, "commit", "-m", "to push").Run()

	// Push via API.
	body, _ := json.Marshal(gen.PushProjectRequest{RemoteBranch: "kf/main"})
	resp, err := http.Post(srv.URL+"/api/projects/sync-push/push", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST push: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d: %v", resp.StatusCode, errBody)
	}

	var pushResult gen.PushResult
	json.NewDecoder(resp.Body).Decode(&pushResult)
	if !pushResult.Success {
		t.Error("expected success=true")
	}
	if pushResult.RemoteBranch != "kf/main" {
		t.Errorf("remote_branch = %q, want %q", pushResult.RemoteBranch, "kf/main")
	}

	// Verify the remote branch exists on the bare repo.
	out, _ := e2eCleanGitCmd("git", "-C", bareDir, "branch", "--list", "kf/main").Output()
	if len(out) == 0 {
		t.Error("remote branch kf/main not found on bare repo")
	}
}

func TestE2E_SyncPull(t *testing.T) {
	srv := startE2EServerWithGitSync(t)
	bareDir, cloneDir := initE2ETestRepo(t)
	registerRealProject(t, srv, "sync-pull", cloneDir, bareDir)

	// Push an upstream change via a second clone.
	dir := filepath.Dir(cloneDir)
	tmpWork := filepath.Join(dir, "tmp-upstream")
	e2eCleanGitCmd("git", "clone", bareDir, tmpWork).Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	if err := os.WriteFile(filepath.Join(tmpWork, "upstream.txt"), []byte("upstream"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	e2eCleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "commit", "-m", "upstream change").Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	// Pull via API.
	body, _ := json.Marshal(gen.PullProjectRequest{})
	resp, err := http.Post(srv.URL+"/api/projects/sync-pull/pull", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST pull: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		t.Fatalf("expected 200, got %d: %v", resp.StatusCode, errBody)
	}

	var pullResult gen.PullResult
	json.NewDecoder(resp.Body).Decode(&pullResult)
	if !pullResult.Success {
		t.Error("expected success=true")
	}
	if pullResult.NewHead == "" {
		t.Error("expected non-empty new_head")
	}

	// Verify the upstream file is now in the clone.
	if _, err := os.Stat(filepath.Join(cloneDir, "upstream.txt")); err != nil {
		t.Error("upstream.txt not found in clone after pull")
	}
}

func TestE2E_SyncStatus_Ahead(t *testing.T) {
	srv := startE2EServerWithGitSync(t)
	bareDir, cloneDir := initE2ETestRepo(t)
	registerRealProject(t, srv, "sync-ahead", cloneDir, bareDir)

	// Make a local commit so the clone is ahead of origin.
	if err := os.WriteFile(filepath.Join(cloneDir, "ahead.txt"), []byte("ahead"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	e2eCleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	e2eCleanGitCmd("git", "-C", cloneDir, "commit", "-m", "local ahead").Run()

	// Check sync status via API.
	resp, err := http.Get(srv.URL + "/api/projects/sync-ahead/sync-status")
	if err != nil {
		t.Fatalf("GET sync-status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var status gen.SyncStatusResponse
	json.NewDecoder(resp.Body).Decode(&status)
	if status.Ahead < 1 {
		t.Errorf("expected ahead >= 1, got %d", status.Ahead)
	}
	if status.Status != gen.Ahead {
		t.Errorf("status = %q, want %q", status.Status, gen.Ahead)
	}
}

func TestE2E_SyncStatus_Behind(t *testing.T) {
	srv := startE2EServerWithGitSync(t)
	bareDir, cloneDir := initE2ETestRepo(t)
	registerRealProject(t, srv, "sync-behind", cloneDir, bareDir)

	// Push an upstream commit via a second clone.
	dir := filepath.Dir(cloneDir)
	tmpWork := filepath.Join(dir, "tmp-behind")
	e2eCleanGitCmd("git", "clone", bareDir, tmpWork).Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	if err := os.WriteFile(filepath.Join(tmpWork, "behind.txt"), []byte("upstream"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	e2eCleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "commit", "-m", "upstream behind").Run()
	e2eCleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	// Fetch so the clone knows about the upstream commit.
	e2eCleanGitCmd("git", "-C", cloneDir, "fetch", "origin").Run()

	// Check sync status via API.
	resp, err := http.Get(srv.URL + "/api/projects/sync-behind/sync-status")
	if err != nil {
		t.Fatalf("GET sync-status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var status gen.SyncStatusResponse
	json.NewDecoder(resp.Body).Decode(&status)
	if status.Behind < 1 {
		t.Errorf("expected behind >= 1, got %d", status.Behind)
	}
	if status.Status != gen.Behind {
		t.Errorf("status = %q, want %q", status.Status, gen.Behind)
	}
}
