package pool

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()

	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:   "worker-1",
				Path:   "/tmp/project/worker-1",
				Branch: "worker-1",
				Status: StatusIdle,
			},
			"worker-2": {
				Name:       "worker-2",
				Path:       "/tmp/project/worker-2",
				Branch:     "worker-2",
				Status:     StatusInUse,
				TrackID:    "auth_20260307",
				AgentID:    "uuid-123",
				AcquiredAt: timePtr(time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)),
			},
		},
	}

	if err := p.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(filepath.Join(dir, "pool.json")); err != nil {
		t.Fatalf("pool.json not found: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.MaxSize != 3 {
		t.Errorf("MaxSize = %d, want 3", loaded.MaxSize)
	}
	if len(loaded.Worktrees) != 2 {
		t.Fatalf("Worktrees count = %d, want 2", len(loaded.Worktrees))
	}

	w1 := loaded.Worktrees["worker-1"]
	if w1.Status != StatusIdle {
		t.Errorf("worker-1 status = %q, want %q", w1.Status, StatusIdle)
	}

	w2 := loaded.Worktrees["worker-2"]
	if w2.Status != StatusInUse {
		t.Errorf("worker-2 status = %q, want %q", w2.Status, StatusInUse)
	}
	if w2.TrackID != "auth_20260307" {
		t.Errorf("worker-2 TrackID = %q, want %q", w2.TrackID, "auth_20260307")
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()

	p, err := Load(dir)
	if err != nil {
		t.Fatalf("Load missing: %v", err)
	}
	if p.MaxSize != DefaultMaxSize {
		t.Errorf("MaxSize = %d, want %d", p.MaxSize, DefaultMaxSize)
	}
	if len(p.Worktrees) != 0 {
		t.Errorf("Worktrees count = %d, want 0", len(p.Worktrees))
	}
}

func TestAcquireIdle(t *testing.T) {
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:   "worker-1",
				Path:   "/tmp/project/worker-1",
				Branch: "worker-1",
				Status: StatusIdle,
			},
			"worker-2": {
				Name:   "worker-2",
				Path:   "/tmp/project/worker-2",
				Branch: "worker-2",
				Status: StatusInUse,
			},
		},
		ProjectRoot: "/tmp/project",
		gitRunner:   &fakeGitRunner{},
	}

	w, err := p.Acquire()
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if w.Name != "worker-1" {
		t.Errorf("got worker %q, want worker-1", w.Name)
	}
	if w.Status != StatusInUse {
		t.Errorf("status = %q, want %q", w.Status, StatusInUse)
	}
	if w.AcquiredAt == nil {
		t.Error("AcquiredAt should be set")
	}
}

func TestAcquireCreatesNew(t *testing.T) {
	p := &Pool{
		MaxSize:     3,
		Worktrees:   map[string]*Worktree{},
		ProjectRoot: "/tmp/project",
		gitRunner:   &fakeGitRunner{},
	}

	w, err := p.Acquire()
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if w.Name != "worker-1" {
		t.Errorf("got worker %q, want worker-1", w.Name)
	}
	if w.Status != StatusInUse {
		t.Errorf("status = %q, want %q", w.Status, StatusInUse)
	}
	if len(p.Worktrees) != 1 {
		t.Errorf("pool size = %d, want 1", len(p.Worktrees))
	}
}

func TestAcquireExhausted(t *testing.T) {
	p := &Pool{
		MaxSize: 1,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:   "worker-1",
				Status: StatusInUse,
			},
		},
		ProjectRoot: "/tmp/project",
		gitRunner:   &fakeGitRunner{},
	}

	_, err := p.Acquire()
	if err == nil {
		t.Fatal("expected error when pool exhausted")
	}
}

type fakeGitRunner struct {
	calls          [][]string
	hasAhead       bool
	hasAheadErr    error
	commitWIPErr   error
	listStash      []string
	listStashErr   error
	mergeBranchErr error
}

func (f *fakeGitRunner) WorktreeAdd(path, branch string) error {
	f.calls = append(f.calls, []string{"worktree", "add", path, branch})
	return nil
}

func (f *fakeGitRunner) WorktreeRemove(path string) error {
	f.calls = append(f.calls, []string{"worktree", "remove", path})
	return nil
}

func (f *fakeGitRunner) ResetHardMain(worktreePath string) error {
	f.calls = append(f.calls, []string{"reset", "--hard", "main", worktreePath})
	return nil
}

func (f *fakeGitRunner) CheckoutBranch(worktreePath, branch string) error {
	f.calls = append(f.calls, []string{"checkout", branch, worktreePath})
	return nil
}

func (f *fakeGitRunner) CreateBranch(worktreePath, branch string) error {
	f.calls = append(f.calls, []string{"checkout", "-b", branch, worktreePath})
	return nil
}

func (f *fakeGitRunner) DeleteBranch(branch string) error {
	f.calls = append(f.calls, []string{"branch", "-D", branch})
	return nil
}

func (f *fakeGitRunner) AddAll(worktreePath string) error {
	f.calls = append(f.calls, []string{"add", "-A", worktreePath})
	return nil
}

func (f *fakeGitRunner) CommitWIP(worktreePath string) error {
	f.calls = append(f.calls, []string{"commit", "wip", worktreePath})
	return f.commitWIPErr
}

func (f *fakeGitRunner) HasCommitsAhead(worktreePath, base string) (bool, error) {
	f.calls = append(f.calls, []string{"log", base + "..HEAD", worktreePath})
	return f.hasAhead, f.hasAheadErr
}

func (f *fakeGitRunner) CreateStashBranch(worktreePath, stashBranch string) error {
	f.calls = append(f.calls, []string{"branch", stashBranch, worktreePath})
	return nil
}

func (f *fakeGitRunner) ListStashBranches(trackID string) ([]string, error) {
	f.calls = append(f.calls, []string{"branch", "--list", "stash/" + trackID + "/*"})
	return f.listStash, f.listStashErr
}

func (f *fakeGitRunner) MergeBranch(worktreePath, branch string) error {
	f.calls = append(f.calls, []string{"merge", branch, worktreePath})
	return f.mergeBranchErr
}

func (f *fakeGitRunner) DeleteBranches(branches []string) error {
	args := append([]string{"branch", "-D"}, branches...)
	f.calls = append(f.calls, args)
	return nil
}

func TestPrepare(t *testing.T) {
	runner := &fakeGitRunner{}
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:   "worker-1",
				Path:   "/tmp/project/worker-1",
				Branch: "worker-1",
				Status: StatusInUse,
			},
		},
		ProjectRoot: "/tmp/project",
		gitRunner:   runner,
	}

	w := p.Worktrees["worker-1"]
	if err := p.Prepare(w, "auth_20260307"); err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	if w.TrackID != "auth_20260307" {
		t.Errorf("TrackID = %q, want %q", w.TrackID, "auth_20260307")
	}

	// Verify git commands were called.
	if len(runner.calls) < 3 {
		t.Fatalf("expected at least 3 git calls, got %d", len(runner.calls))
	}
	// Should: checkout pool branch, reset --hard main, create impl branch
	if runner.calls[0][0] != "checkout" {
		t.Errorf("first call should be checkout, got %v", runner.calls[0])
	}
	if runner.calls[1][0] != "reset" {
		t.Errorf("second call should be reset, got %v", runner.calls[1])
	}
	if runner.calls[2][0] != "checkout" && runner.calls[2][1] != "-b" {
		t.Errorf("third call should be checkout -b, got %v", runner.calls[2])
	}
}

func TestReturn(t *testing.T) {
	runner := &fakeGitRunner{}
	now := time.Now()
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:       "worker-1",
				Path:       "/tmp/project/worker-1",
				Branch:     "worker-1",
				Status:     StatusInUse,
				TrackID:    "auth_20260307",
				AgentID:    "uuid-123",
				AcquiredAt: &now,
			},
		},
		ProjectRoot: "/tmp/project",
		gitRunner:   runner,
	}

	w := p.Worktrees["worker-1"]
	if err := p.Return(w); err != nil {
		t.Fatalf("Return: %v", err)
	}

	if w.Status != StatusIdle {
		t.Errorf("status = %q, want %q", w.Status, StatusIdle)
	}
	if w.TrackID != "" {
		t.Errorf("TrackID should be empty, got %q", w.TrackID)
	}
	if w.AgentID != "" {
		t.Errorf("AgentID should be empty, got %q", w.AgentID)
	}
	if w.AcquiredAt != nil {
		t.Error("AcquiredAt should be nil")
	}
}

func TestStatus(t *testing.T) {
	now := time.Now()
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {Name: "worker-1", Status: StatusIdle},
			"worker-2": {Name: "worker-2", Status: StatusInUse, TrackID: "track-1", AcquiredAt: &now},
		},
	}

	statuses := p.Status()
	if len(statuses) != 2 {
		t.Fatalf("Status count = %d, want 2", len(statuses))
	}
	// Should be sorted by name.
	if statuses[0].Name != "worker-1" {
		t.Errorf("first = %q, want worker-1", statuses[0].Name)
	}
	if statuses[1].Name != "worker-2" {
		t.Errorf("second = %q, want worker-2", statuses[1].Name)
	}
}

func TestStash_CreatesStashBranch(t *testing.T) {
	runner := &fakeGitRunner{hasAhead: true}
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:    "worker-1",
				Path:    "/tmp/project/worker-1",
				Branch:  "worker-1",
				Status:  StatusInUse,
				TrackID: "auth_20260307",
			},
		},
		gitRunner: runner,
	}

	w := p.Worktrees["worker-1"]
	stashBranch, err := p.Stash(w)
	if err != nil {
		t.Fatalf("Stash: %v", err)
	}
	if stashBranch != "stash/auth_20260307/worker-1" {
		t.Errorf("stashBranch = %q, want %q", stashBranch, "stash/auth_20260307/worker-1")
	}

	// Verify git calls: add -A, commit wip, hasCommitsAhead, branch stash
	hasAdd := false
	hasCommit := false
	hasBranch := false
	for _, c := range runner.calls {
		if c[0] == "add" {
			hasAdd = true
		}
		if c[0] == "commit" {
			hasCommit = true
		}
		if c[0] == "branch" && len(c) > 1 && c[1] == "stash/auth_20260307/worker-1" {
			hasBranch = true
		}
	}
	if !hasAdd {
		t.Error("expected AddAll call")
	}
	if !hasCommit {
		t.Error("expected CommitWIP call")
	}
	if !hasBranch {
		t.Error("expected CreateStashBranch call")
	}
}

func TestStash_NoCommitsAhead(t *testing.T) {
	runner := &fakeGitRunner{hasAhead: false}
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:    "worker-1",
				Path:    "/tmp/project/worker-1",
				Branch:  "worker-1",
				Status:  StatusInUse,
				TrackID: "auth_20260307",
			},
		},
		gitRunner: runner,
	}

	w := p.Worktrees["worker-1"]
	stashBranch, err := p.Stash(w)
	if err != nil {
		t.Fatalf("Stash: %v", err)
	}
	if stashBranch != "" {
		t.Errorf("expected empty stash branch when nothing ahead, got %q", stashBranch)
	}
}

func TestStash_NoTrackID(t *testing.T) {
	p := &Pool{
		Worktrees: map[string]*Worktree{
			"worker-1": {Name: "worker-1", Path: "/tmp/w1", Branch: "worker-1", Status: StatusInUse},
		},
		gitRunner: &fakeGitRunner{},
	}
	w := p.Worktrees["worker-1"]
	stashBranch, err := p.Stash(w)
	if err != nil {
		t.Fatalf("Stash: %v", err)
	}
	if stashBranch != "" {
		t.Errorf("expected empty stash for no trackID, got %q", stashBranch)
	}
}

func TestReturn_StashesBeforeCleanup(t *testing.T) {
	runner := &fakeGitRunner{hasAhead: true}
	now := time.Now()
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:       "worker-1",
				Path:       "/tmp/project/worker-1",
				Branch:     "worker-1",
				Status:     StatusInUse,
				TrackID:    "auth_20260307",
				AgentID:    "uuid-123",
				AcquiredAt: &now,
			},
		},
		gitRunner: runner,
	}

	w := p.Worktrees["worker-1"]
	if err := p.Return(w); err != nil {
		t.Fatalf("Return: %v", err)
	}

	// Verify stash branch was created before cleanup.
	hasStashCreate := false
	for _, c := range runner.calls {
		if c[0] == "branch" && len(c) > 1 && c[1] == "stash/auth_20260307/worker-1" {
			hasStashCreate = true
		}
	}
	if !hasStashCreate {
		t.Error("expected stash branch creation during Return")
	}

	// Verify worktree is idle after return.
	if w.Status != StatusIdle {
		t.Errorf("status = %q, want %q", w.Status, StatusIdle)
	}
}

func TestPrepare_MergesStash(t *testing.T) {
	runner := &fakeGitRunner{
		listStash: []string{"stash/auth_20260307/worker-2"},
	}
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:   "worker-1",
				Path:   "/tmp/project/worker-1",
				Branch: "worker-1",
				Status: StatusInUse,
			},
		},
		gitRunner: runner,
	}

	w := p.Worktrees["worker-1"]
	if err := p.Prepare(w, "auth_20260307"); err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	// Verify merge was called.
	hasMerge := false
	hasDeleteBranches := false
	for _, c := range runner.calls {
		if c[0] == "merge" && c[1] == "stash/auth_20260307/worker-2" {
			hasMerge = true
		}
		if c[0] == "branch" && c[1] == "-D" {
			hasDeleteBranches = true
		}
	}
	if !hasMerge {
		t.Error("expected merge of stash branch during Prepare")
	}
	if !hasDeleteBranches {
		t.Error("expected deletion of stash branches after merge")
	}
	if w.TrackID != "auth_20260307" {
		t.Errorf("TrackID = %q, want %q", w.TrackID, "auth_20260307")
	}
}

func TestPrepare_NoStash(t *testing.T) {
	runner := &fakeGitRunner{}
	p := &Pool{
		MaxSize: 3,
		Worktrees: map[string]*Worktree{
			"worker-1": {
				Name:   "worker-1",
				Path:   "/tmp/project/worker-1",
				Branch: "worker-1",
				Status: StatusInUse,
			},
		},
		gitRunner: runner,
	}

	w := p.Worktrees["worker-1"]
	if err := p.Prepare(w, "auth_20260307"); err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	// Verify no merge was called.
	for _, c := range runner.calls {
		if c[0] == "merge" {
			t.Error("unexpected merge call when no stash exists")
		}
	}
}

func TestCleanupStash(t *testing.T) {
	runner := &fakeGitRunner{
		listStash: []string{"stash/auth_20260307/worker-1", "stash/auth_20260307/worker-2"},
	}
	p := &Pool{gitRunner: runner}

	if err := p.CleanupStash("auth_20260307"); err != nil {
		t.Fatalf("CleanupStash: %v", err)
	}

	// Verify delete was called with both branches.
	hasDelete := false
	for _, c := range runner.calls {
		if c[0] == "branch" && c[1] == "-D" {
			hasDelete = true
			if len(c) != 4 {
				t.Errorf("expected 4 args in delete call, got %d: %v", len(c), c)
			}
		}
	}
	if !hasDelete {
		t.Error("expected DeleteBranches call")
	}
}

func TestCleanupStash_NoBranches(t *testing.T) {
	runner := &fakeGitRunner{}
	p := &Pool{gitRunner: runner}

	if err := p.CleanupStash("nonexistent_track"); err != nil {
		t.Fatalf("CleanupStash: %v", err)
	}

	// Verify no delete was called.
	for _, c := range runner.calls {
		if c[0] == "branch" && len(c) > 1 && c[1] == "-D" {
			t.Error("unexpected DeleteBranches call when no stash exists")
		}
	}
}

func TestEnsureSkillsSymlink_CreatesSymlink(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	worktreeDir := t.TempDir()

	// Create skills in project dir.
	srcSkills := filepath.Join(projectDir, ".claude", "skills")
	os.MkdirAll(filepath.Join(srcSkills, "kf-developer"), 0o755)
	os.WriteFile(filepath.Join(srcSkills, "kf-developer", "SKILL.md"), []byte("# Dev"), 0o644)

	if err := EnsureSkillsSymlink(worktreeDir, projectDir); err != nil {
		t.Fatalf("EnsureSkillsSymlink: %v", err)
	}

	destSkills := filepath.Join(worktreeDir, ".claude", "skills")
	target, err := os.Readlink(destSkills)
	if err != nil {
		t.Fatalf("expected symlink, got: %v", err)
	}
	if target != srcSkills {
		t.Errorf("symlink target = %q, want %q", target, srcSkills)
	}

	// Verify skill is accessible via symlink.
	if _, err := os.Stat(filepath.Join(destSkills, "kf-developer", "SKILL.md")); err != nil {
		t.Errorf("skill not accessible via symlink: %v", err)
	}
}

func TestEnsureSkillsSymlink_NoOpWhenSameDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	srcSkills := filepath.Join(dir, ".claude", "skills")
	os.MkdirAll(srcSkills, 0o755)

	if err := EnsureSkillsSymlink(dir, dir); err != nil {
		t.Fatalf("EnsureSkillsSymlink: %v", err)
	}

	// Should not create a symlink to itself.
	if _, err := os.Readlink(srcSkills); err == nil {
		t.Error("should not create symlink when dirs are the same")
	}
}

func TestEnsureSkillsSymlink_NoOpWhenNoProjectSkills(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	worktreeDir := t.TempDir()

	// Project has no .claude/skills/ — should be a no-op.
	if err := EnsureSkillsSymlink(worktreeDir, projectDir); err != nil {
		t.Fatalf("EnsureSkillsSymlink: %v", err)
	}

	destSkills := filepath.Join(worktreeDir, ".claude", "skills")
	if _, err := os.Stat(destSkills); !os.IsNotExist(err) {
		t.Error("should not create anything when project has no skills")
	}
}

func TestEnsureSkillsSymlink_IdempotentCorrectTarget(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	worktreeDir := t.TempDir()

	srcSkills := filepath.Join(projectDir, ".claude", "skills")
	os.MkdirAll(srcSkills, 0o755)

	// Create symlink twice — second call should be a no-op.
	if err := EnsureSkillsSymlink(worktreeDir, projectDir); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := EnsureSkillsSymlink(worktreeDir, projectDir); err != nil {
		t.Fatalf("second call: %v", err)
	}

	target, err := os.Readlink(filepath.Join(worktreeDir, ".claude", "skills"))
	if err != nil {
		t.Fatalf("expected symlink: %v", err)
	}
	if target != srcSkills {
		t.Errorf("target = %q, want %q", target, srcSkills)
	}
}

func TestEnsureSkillsSymlink_SkipsRealDirectory(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	worktreeDir := t.TempDir()

	srcSkills := filepath.Join(projectDir, ".claude", "skills")
	os.MkdirAll(srcSkills, 0o755)

	// Create a real skills directory in the worktree.
	destSkills := filepath.Join(worktreeDir, ".claude", "skills")
	os.MkdirAll(destSkills, 0o755)

	if err := EnsureSkillsSymlink(worktreeDir, projectDir); err != nil {
		t.Fatalf("EnsureSkillsSymlink: %v", err)
	}

	// Should not replace the real directory with a symlink.
	if _, err := os.Readlink(destSkills); err == nil {
		t.Error("should not replace real directory with symlink")
	}
}

func TestEnsureSkillsSymlink_FixesWrongTarget(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	worktreeDir := t.TempDir()
	otherDir := t.TempDir()

	srcSkills := filepath.Join(projectDir, ".claude", "skills")
	os.MkdirAll(srcSkills, 0o755)

	// Create symlink to wrong target.
	destClaudeDir := filepath.Join(worktreeDir, ".claude")
	os.MkdirAll(destClaudeDir, 0o755)
	os.Symlink(filepath.Join(otherDir, "skills"), filepath.Join(destClaudeDir, "skills"))

	if err := EnsureSkillsSymlink(worktreeDir, projectDir); err != nil {
		t.Fatalf("EnsureSkillsSymlink: %v", err)
	}

	target, err := os.Readlink(filepath.Join(destClaudeDir, "skills"))
	if err != nil {
		t.Fatalf("expected symlink: %v", err)
	}
	if target != srcSkills {
		t.Errorf("target = %q, want %q", target, srcSkills)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
