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
	calls [][]string
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

func timePtr(t time.Time) *time.Time {
	return &t
}
