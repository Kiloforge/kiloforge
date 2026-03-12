package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// cleanGitEnv returns os.Environ() with GIT_DIR and GIT_WORK_TREE removed
// to prevent worktree env vars from leaking into subprocess git operations.
func cleanGitEnv() []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_DIR=") || strings.HasPrefix(e, "GIT_WORK_TREE=") {
			continue
		}
		env = append(env, e)
	}
	return env
}

// cleanGitCmd creates a git exec.Cmd with GIT_DIR/GIT_WORK_TREE removed.
func cleanGitCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = cleanGitEnv()
	return cmd
}

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
		cmd.Env = cleanGitEnv()
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
	cmd := cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com")
	cmd.Run()
	cmd = cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test")
	cmd.Run()

	// Create initial commit.
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	// Clone for the test.
	run("git", "clone", bareDir, cloneDir)
	cleanGitCmd("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", cloneDir, "config", "user.name", "Test").Run()

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
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "local change").Run()

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
		cmd.Env = cleanGitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	// Create bare repo and clone.
	run("git", "init", "--bare", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	run("git", "clone", bareDir, cloneDir)
	cleanGitCmd("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", cloneDir, "config", "user.name", "Test").Run()

	// Push a new commit to origin via tmpWork.
	f2, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f2.WriteString("upstream change")
	f2.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "upstream change").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

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
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "to push").Run()

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
	out, err := cleanGitCmd("git", "-C", bareDir, "branch", "--list", "kf/main").Output()
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
	cleanGitCmd("git", "-C", dir, "init").Run()

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
		cmd.Env = cleanGitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	run("git", "init", "--bare", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	run("git", "clone", bareDir, cloneDir)
	cleanGitCmd("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", cloneDir, "config", "user.name", "Test").Run()

	// Push upstream changes.
	f2, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f2.WriteString("upstream change")
	f2.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "upstream").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	gs := New()
	result, err := gs.PullFromRemote(context.Background(), cloneDir, "main", "", "")
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
		cmd.Env = cleanGitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	run("git", "init", "--bare", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	run("git", "clone", bareDir, cloneDir)
	cleanGitCmd("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", cloneDir, "config", "user.name", "Test").Run()

	// Create divergence: local commit.
	f2, _ := os.Create(filepath.Join(cloneDir, "local.txt"))
	f2.WriteString("local")
	f2.Close()
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "local").Run()

	// Create divergence: upstream commit.
	f3, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f3.WriteString("upstream")
	f3.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "upstream").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	gs := New()
	_, err := gs.PullFromRemote(context.Background(), cloneDir, "main", "", "")
	if err == nil {
		t.Error("expected error for diverged repo")
	}
	if !strings.Contains(err.Error(), "diverged") {
		t.Errorf("expected diverged error, got: %v", err)
	}
}

func TestPushToRemote_Conflict(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = cleanGitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	run("git", "init", "--bare", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	run("git", "clone", bareDir, cloneDir)
	cleanGitCmd("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", cloneDir, "config", "user.name", "Test").Run()

	// Push a commit from clone to create the remote branch kf/main.
	f2, _ := os.Create(filepath.Join(cloneDir, "first.txt"))
	f2.WriteString("first push")
	f2.Close()
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "first push").Run()

	gs := New()
	_, err := gs.PushToRemote(context.Background(), cloneDir, "main", "kf/main", "")
	if err != nil {
		t.Fatalf("initial push failed: %v", err)
	}

	// Create divergence on the kf/main remote branch:
	// Make a separate commit in tmpWork and force push to kf/main.
	f3, _ := os.Create(filepath.Join(tmpWork, "diverge.txt"))
	f3.WriteString("diverging commit")
	f3.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "diverge kf/main").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main:refs/heads/kf/main", "--force").Run()

	// Make another local commit in clone so local main diverges from origin/kf/main.
	f4, _ := os.Create(filepath.Join(cloneDir, "second.txt"))
	f4.WriteString("second push")
	f4.Close()
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "second push").Run()

	// This push should fail with ErrSyncConflict.
	_, err = gs.PushToRemote(context.Background(), cloneDir, "main", "kf/main", "")
	if err == nil {
		t.Fatal("expected error for conflicting push")
	}

	conflict := IsErrSyncConflict(err)
	if conflict == nil {
		t.Fatalf("expected ErrSyncConflict, got: %T: %v", err, err)
	}
	if conflict.Direction != "push" {
		t.Errorf("direction = %q, want %q", conflict.Direction, "push")
	}
	if !strings.Contains(conflict.Error(), "diverged") {
		t.Errorf("error message should contain 'diverged', got: %s", conflict.Error())
	}
}

func TestPullFromRemote_Diverged_ErrSyncConflict(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = cleanGitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	run("git", "init", "--bare", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	run("git", "clone", bareDir, cloneDir)
	cleanGitCmd("git", "-C", cloneDir, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", cloneDir, "config", "user.name", "Test").Run()

	// Create divergence.
	f2, _ := os.Create(filepath.Join(cloneDir, "local.txt"))
	f2.WriteString("local")
	f2.Close()
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "local").Run()

	f3, _ := os.Create(filepath.Join(tmpWork, "upstream.txt"))
	f3.WriteString("upstream")
	f3.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "upstream").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "main").Run()

	gs := New()
	_, err := gs.PullFromRemote(context.Background(), cloneDir, "main", "", "")
	if err == nil {
		t.Fatal("expected error for diverged branches")
	}

	conflict := IsErrSyncConflict(err)
	if conflict == nil {
		t.Fatalf("expected ErrSyncConflict, got: %T: %v", err, err)
	}
	if conflict.Direction != "pull" {
		t.Errorf("direction = %q, want %q", conflict.Direction, "pull")
	}
	if conflict.Ahead != 1 {
		t.Errorf("ahead = %d, want 1", conflict.Ahead)
	}
	if conflict.Behind != 1 {
		t.Errorf("behind = %d, want 1", conflict.Behind)
	}
}

func TestIsErrSyncConflict_Nil(t *testing.T) {
	t.Parallel()
	if IsErrSyncConflict(nil) != nil {
		t.Error("expected nil for nil error")
	}
	if IsErrSyncConflict(fmt.Errorf("some other error")) != nil {
		t.Error("expected nil for non-ErrSyncConflict error")
	}
}

func TestCreateMirrorClone(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	mirrorDir := filepath.Join(t.TempDir(), "mirror")

	gs := New()
	err := gs.CreateMirrorClone(context.Background(), cloneDir, mirrorDir)
	if err != nil {
		t.Fatalf("CreateMirrorClone: %v", err)
	}

	// Verify the mirror has the same HEAD as source.
	srcHead, _ := cleanGitCmd("git", "-C", cloneDir, "rev-parse", "HEAD").Output()
	mirrorHead, _ := cleanGitCmd("git", "-C", mirrorDir, "rev-parse", "HEAD").Output()
	if strings.TrimSpace(string(srcHead)) != strings.TrimSpace(string(mirrorHead)) {
		t.Errorf("mirror HEAD %q != source HEAD %q", string(mirrorHead), string(srcHead))
	}
}

func TestForcePushToMirror(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	mirrorDir := filepath.Join(t.TempDir(), "mirror")

	gs := New()
	// Create mirror first.
	if err := gs.CreateMirrorClone(context.Background(), cloneDir, mirrorDir); err != nil {
		t.Fatalf("CreateMirrorClone: %v", err)
	}

	// Make a new commit in the source.
	f, _ := os.Create(filepath.Join(cloneDir, "new.txt"))
	f.WriteString("new content")
	f.Close()
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "new commit").Run()

	// Force push to mirror.
	if err := gs.ForcePushToMirror(context.Background(), cloneDir, mirrorDir, ""); err != nil {
		t.Fatalf("ForcePushToMirror: %v", err)
	}

	// Verify mirror now has the new commit.
	srcHead, _ := cleanGitCmd("git", "-C", cloneDir, "rev-parse", "HEAD").Output()
	mirrorHead, _ := cleanGitCmd("git", "-C", mirrorDir, "rev-parse", "HEAD").Output()
	if strings.TrimSpace(string(srcHead)) != strings.TrimSpace(string(mirrorHead)) {
		t.Errorf("mirror HEAD %q != source HEAD %q after force push", string(mirrorHead), string(srcHead))
	}
}

func TestForcePushToMirror_MirrorNotExist(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	gs := New()
	err := gs.ForcePushToMirror(context.Background(), cloneDir, "/nonexistent/mirror", "")
	if err == nil {
		t.Fatal("expected error when mirror does not exist")
	}
}

func TestForcePushToMirror_EmptyRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	sourceDir := filepath.Join(dir, "source")
	mirrorDir := filepath.Join(dir, "mirror")

	// Create an empty repo (no commits).
	cleanGitCmd("git", "init", sourceDir).Run()
	cleanGitCmd("git", "clone", "file://"+sourceDir, mirrorDir).Run()

	gs := New()
	err := gs.ForcePushToMirror(context.Background(), sourceDir, mirrorDir, "")
	// Should fail — there's no main branch to push.
	if err == nil {
		t.Fatal("expected error for empty repo force push")
	}
}

func TestMirrorIntegration_CloneCommitSyncVerify(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	mirrorDir := filepath.Join(t.TempDir(), "mirror")

	gs := New()

	// Step 1: Create mirror clone.
	if err := gs.CreateMirrorClone(context.Background(), cloneDir, mirrorDir); err != nil {
		t.Fatalf("CreateMirrorClone: %v", err)
	}

	// Step 2: Verify initial content matches.
	srcFiles, _ := os.ReadDir(cloneDir)
	mirrorFiles, _ := os.ReadDir(mirrorDir)
	// Filter out .git dirs for comparison.
	srcCount := countNonGit(srcFiles)
	mirrorCount := countNonGit(mirrorFiles)
	if srcCount != mirrorCount {
		t.Errorf("initial file count: source=%d, mirror=%d", srcCount, mirrorCount)
	}

	// Step 3: Add a new file, commit to source.
	f, _ := os.Create(filepath.Join(cloneDir, "feature.txt"))
	f.WriteString("new feature")
	f.Close()
	cleanGitCmd("git", "-C", cloneDir, "add", ".").Run()
	cleanGitCmd("git", "-C", cloneDir, "commit", "-m", "add feature").Run()

	// Step 4: Force push to mirror.
	if err := gs.ForcePushToMirror(context.Background(), cloneDir, mirrorDir, ""); err != nil {
		t.Fatalf("ForcePushToMirror: %v", err)
	}

	// Step 5: Verify mirror has the new file.
	content, err := os.ReadFile(filepath.Join(mirrorDir, "feature.txt"))
	if err != nil {
		t.Fatalf("feature.txt not in mirror: %v", err)
	}
	if string(content) != "new feature" {
		t.Errorf("mirror content = %q, want %q", content, "new feature")
	}

	// Step 6: Verify HEAD matches.
	srcHead, _ := cleanGitCmd("git", "-C", cloneDir, "rev-parse", "HEAD").Output()
	mirrorHead, _ := cleanGitCmd("git", "-C", mirrorDir, "rev-parse", "HEAD").Output()
	if strings.TrimSpace(string(srcHead)) != strings.TrimSpace(string(mirrorHead)) {
		t.Errorf("HEAD mismatch after sync: source=%s mirror=%s", srcHead, mirrorHead)
	}
}

func countNonGit(entries []os.DirEntry) int {
	n := 0
	for _, e := range entries {
		if e.Name() != ".git" {
			n++
		}
	}
	return n
}

func TestDetectDefaultBranch_Main(t *testing.T) {
	t.Parallel()
	_, cloneDir := initBareAndClone(t)

	gs := New()
	branch := gs.DetectDefaultBranch(context.Background(), cloneDir)
	if branch != "main" {
		t.Errorf("DetectDefaultBranch = %q, want %q", branch, "main")
	}
}

func TestDetectDefaultBranch_Master(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = cleanGitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	// Create bare repo with "master" as default branch.
	run("git", "init", "--bare", "--initial-branch=master", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "master").Run()

	run("git", "clone", bareDir, cloneDir)

	gs := New()
	branch := gs.DetectDefaultBranch(context.Background(), cloneDir)
	if branch != "master" {
		t.Errorf("DetectDefaultBranch = %q, want %q", branch, "master")
	}
}

func TestDetectDefaultBranch_Custom(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bareDir := filepath.Join(dir, "origin.git")
	cloneDir := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = cleanGitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	// Create bare repo with "develop" as default branch.
	run("git", "init", "--bare", "--initial-branch=develop", bareDir)
	tmpWork := filepath.Join(dir, "tmp-work")
	run("git", "clone", bareDir, tmpWork)
	cleanGitCmd("git", "-C", tmpWork, "config", "user.email", "test@test.com").Run()
	cleanGitCmd("git", "-C", tmpWork, "config", "user.name", "Test").Run()
	f, _ := os.Create(filepath.Join(tmpWork, "README.md"))
	f.WriteString("# test")
	f.Close()
	cleanGitCmd("git", "-C", tmpWork, "add", ".").Run()
	cleanGitCmd("git", "-C", tmpWork, "commit", "-m", "initial").Run()
	cleanGitCmd("git", "-C", tmpWork, "push", "origin", "develop").Run()

	run("git", "clone", bareDir, cloneDir)

	gs := New()
	branch := gs.DetectDefaultBranch(context.Background(), cloneDir)
	if branch != "develop" {
		t.Errorf("DetectDefaultBranch = %q, want %q", branch, "develop")
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
