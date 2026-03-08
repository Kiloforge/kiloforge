package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initBareAndClone creates a bare "origin" repo and a clone of it.
// Returns (bareDir, cloneDir). The clone has one initial commit on main.
func initBareAndClone(t *testing.T) (string, string) {
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
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	// Create bare repo.
	run("git", "init", "--bare", bareDir)

	// Create a temporary working dir to make an initial commit.
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)

	// Configure git user in tmpWork.
	cmd := exec.Command("git", "-C", tmpWork, "config", "user.email", "test@test.com")
	cmd.Run()
	cmd = exec.Command("git", "-C", tmpWork, "config", "user.name", "Test")
	cmd.Run()

	// Create initial commit.
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	exec.Command("git", "-C", tmpWork, "add", ".").Run()
	exec.Command("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	exec.Command("git", "-C", tmpWork, "push", "origin", "main").Run()

	// Clone for the test.
	run("git", "clone", bareDir, cloneDir)
	exec.Command("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", cloneDir, "config", "user.name", "Test").Run()

	return bareDir, cloneDir
}

func TestSyncStatus_Synced(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	gs := New()
	status, err := gs.SyncStatus(context.Background(), cloneDir, "")
	if err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if status.Ahead != 0 {
		t.Errorf("ahead = %d, want 0", status.Ahead)
	}
	if status.Behind != 0 {
		t.Errorf("behind = %d, want 0", status.Behind)
	}
	if status.Status != StatusSynced {
		t.Errorf("status = %q, want %q", status.Status, StatusSynced)
	}
	if status.LocalBranch != "main" {
		t.Errorf("local_branch = %q, want %q", status.LocalBranch, "main")
	}
}

func TestSyncStatus_Ahead(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	// Make a local commit.
	f, _ := os.Create(filepath.Join(cloneDir, "new.txt"))
	f.WriteString("new file")
	f.Close()
	exec.Command("git", "-C", cloneDir, "add", ".").Run()
	exec.Command("git", "-C", cloneDir, "commit", "-m", "local change").Run()

	gs := New()
	status, err := gs.SyncStatus(context.Background(), cloneDir, "")
	if err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if status.Ahead != 1 {
		t.Errorf("ahead = %d, want 1", status.Ahead)
	}
	if status.Behind != 0 {
		t.Errorf("behind = %d, want 0", status.Behind)
	}
	if status.Status != StatusAhead {
		t.Errorf("status = %q, want %q", status.Status, StatusAhead)
	}
}

func TestSyncStatus_Behind(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	// Create bare repo and clone.
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

	// Push a new commit to origin via tmpWork.
	f2, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f2.WriteString("upstream change")
	f2.Close()
	exec.Command("git", "-C", tmpWork, "add", ".").Run()
	exec.Command("git", "-C", tmpWork, "commit", "-m", "upstream change").Run()
	exec.Command("git", "-C", tmpWork, "push", "origin", "main").Run()

	// Fetch in clone so it knows about the upstream commit.
	gs := New()
	status, err := gs.SyncStatus(context.Background(), cloneDir, "")
	if err != nil {
		t.Fatalf("SyncStatus: %v", err)
	}
	if status.Behind != 1 {
		t.Errorf("behind = %d, want 1", status.Behind)
	}
	if status.Status != StatusBehind {
		t.Errorf("status = %q, want %q", status.Status, StatusBehind)
	}
}

func TestPushToRemote(t *testing.T) {
	t.Parallel()
	bareDir, cloneDir := initBareAndClone(t)

	// Make a local commit.
	f, _ := os.Create(filepath.Join(cloneDir, "pushed.txt"))
	f.WriteString("push me")
	f.Close()
	exec.Command("git", "-C", cloneDir, "add", ".").Run()
	exec.Command("git", "-C", cloneDir, "commit", "-m", "to push").Run()

	gs := New()
	result, err := gs.PushToRemote(context.Background(), cloneDir, "main", "kf/main", "")
	if err != nil {
		t.Fatalf("PushToRemote: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.RemoteBranch != "kf/main" {
		t.Errorf("remote_branch = %q, want %q", result.RemoteBranch, "kf/main")
	}

	// Verify the remote branch exists.
	out, err := exec.Command("git", "-C", bareDir, "branch", "--list", "kf/main").Output()
	if err != nil {
		t.Fatalf("list branches: %v", err)
	}
	if !strings.Contains(string(out), "kf/main") {
		t.Errorf("remote branch kf/main not found, got: %s", out)
	}
}

func TestPushToRemote_NoOrigin(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	exec.Command("git", "-C", dir, "init").Run()

	gs := New()
	_, err := gs.PushToRemote(context.Background(), dir, "main", "kf/main", "")
	if err == nil {
		t.Error("expected error for repo without origin")
	}
}

func TestPullFromRemote(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
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

	// Push upstream changes.
	f2, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f2.WriteString("upstream change")
	f2.Close()
	exec.Command("git", "-C", tmpWork, "add", ".").Run()
	exec.Command("git", "-C", tmpWork, "commit", "-m", "upstream").Run()
	exec.Command("git", "-C", tmpWork, "push", "origin", "main").Run()

	gs := New()
	result, err := gs.PullFromRemote(context.Background(), cloneDir, "main", "")
	if err != nil {
		t.Fatalf("PullFromRemote: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}

	// Verify the file exists in clone.
	if _, err := os.Stat(filepath.Join(cloneDir, "upstream.txt")); err != nil {
		t.Errorf("upstream.txt not found after pull: %v", err)
	}
}

func TestPullFromRemote_Diverged(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
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

	// Create divergence: local commit.
	f2, _ := os.Create(filepath.Join(cloneDir, "local.txt"))
	f2.WriteString("local")
	f2.Close()
	exec.Command("git", "-C", cloneDir, "add", ".").Run()
	exec.Command("git", "-C", cloneDir, "commit", "-m", "local").Run()

	// Create divergence: upstream commit.
	f3, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f3.WriteString("upstream")
	f3.Close()
	exec.Command("git", "-C", tmpWork, "add", ".").Run()
	exec.Command("git", "-C", tmpWork, "commit", "-m", "upstream").Run()
	exec.Command("git", "-C", tmpWork, "push", "origin", "main").Run()

	gs := New()
	_, err := gs.PullFromRemote(context.Background(), cloneDir, "main", "")
	if err == nil {
		t.Error("expected error for diverged repo")
	}
	if !strings.Contains(err.Error(), "diverged") {
		t.Errorf("expected diverged error, got: %v", err)
	}
}

func TestFetchOrigin(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	gs := New()
	err := gs.FetchOrigin(context.Background(), cloneDir, "")
	if err != nil {
		t.Fatalf("FetchOrigin: %v", err)
	}
}
