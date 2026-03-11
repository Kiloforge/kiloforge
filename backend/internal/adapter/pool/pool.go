package pool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	StatusIdle  = "idle"
	StatusInUse = "in-use"

	DefaultMaxSize = 3
	poolFileName   = "pool.json"
)

// Worktree represents a single worktree slot in the pool.
type Worktree struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Branch     string     `json:"branch"`
	Status     string     `json:"status"`
	TrackID    string     `json:"track_id,omitempty"`
	AgentID    string     `json:"agent_id,omitempty"`
	AcquiredAt *time.Time `json:"acquired_at,omitempty"`
}

// Pool manages a set of git worktrees for developer agents.
type Pool struct {
	Worktrees   map[string]*Worktree `json:"worktrees"`
	MaxSize     int                  `json:"max_size"`
	ProjectRoot string               `json:"-"`
	gitRunner   GitRunner            `json:"-"`
}

// Load reads pool state from the data directory.
// Returns a default empty pool if the file does not exist.
func Load(dataDir string) (*Pool, error) {
	path := filepath.Join(dataDir, poolFileName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Pool{
			Worktrees: map[string]*Worktree{},
			MaxSize:   DefaultMaxSize,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read pool state: %w", err)
	}

	var p Pool
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse pool state: %w", err)
	}
	if p.Worktrees == nil {
		p.Worktrees = map[string]*Worktree{}
	}
	return &p, nil
}

func (p *Pool) git() GitRunner {
	if p.gitRunner != nil {
		return p.gitRunner
	}
	return &execGitRunner{}
}

// Acquire finds an idle worktree or creates a new one if under max size.
// Returns an error if all worktrees are in use and pool is at capacity.
func (p *Pool) Acquire() (*Worktree, error) {
	// Find first idle worktree (sorted for determinism).
	for _, name := range sortedKeys(p.Worktrees) {
		w := p.Worktrees[name]
		if w.Status == StatusIdle {
			now := time.Now().UTC().Truncate(time.Second)
			w.Status = StatusInUse
			w.AcquiredAt = &now
			return w, nil
		}
	}

	// Create a new one if under max.
	if len(p.Worktrees) >= p.MaxSize {
		return nil, fmt.Errorf("pool exhausted: all %d worktrees are in use", p.MaxSize)
	}

	name := fmt.Sprintf("worker-%d", len(p.Worktrees)+1)
	path := filepath.Join(p.ProjectRoot, name)

	if err := p.git().WorktreeAdd(path, name); err != nil {
		return nil, fmt.Errorf("create worktree %s: %w", name, err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	w := &Worktree{
		Name:       name,
		Path:       path,
		Branch:     name,
		Status:     StatusInUse,
		AcquiredAt: &now,
	}
	p.Worktrees[name] = w
	return w, nil
}

// Prepare resets a worktree to main and creates an implementation branch for the given track.
// If stash branches exist for the track, they are merged into the new impl branch.
func (p *Pool) Prepare(w *Worktree, trackID string) error {
	g := p.git()

	// Checkout the pool branch.
	if err := g.CheckoutBranch(w.Path, w.Branch); err != nil {
		return fmt.Errorf("checkout pool branch %s: %w", w.Branch, err)
	}

	// Reset to main.
	if err := g.ResetHardMain(w.Path); err != nil {
		return fmt.Errorf("reset to main: %w", err)
	}

	// Create implementation branch.
	if err := g.CreateBranch(w.Path, trackID); err != nil {
		return fmt.Errorf("create branch %s: %w", trackID, err)
	}

	// Check for stash branches and merge if found.
	stashes, err := g.ListStashBranches(trackID)
	if err == nil && len(stashes) > 0 {
		for _, stash := range stashes {
			if mergeErr := g.MergeBranch(w.Path, stash); mergeErr != nil {
				// Stash merge failure is not fatal — leave the stash for manual recovery.
				continue
			}
		}
		// Delete merged stash branches.
		_ = g.DeleteBranches(stashes)
	}

	// Ensure skills are discoverable from the worktree.
	if p.ProjectRoot != "" {
		_ = EnsureSkillsSymlink(w.Path, p.ProjectRoot)
	}

	w.TrackID = trackID
	return nil
}

// Stash auto-commits uncommitted changes and creates a stash branch
// for the worktree's current track. Returns the stash branch name if one
// was created, or empty string if there was nothing to stash.
func (p *Pool) Stash(w *Worktree) (string, error) {
	if w.TrackID == "" {
		return "", nil
	}
	g := p.git()

	// Auto-commit any uncommitted changes (best effort — nothing to commit is fine).
	_ = g.AddAll(w.Path)
	_ = g.CommitWIP(w.Path)

	// Check if impl branch has commits ahead of main.
	ahead, err := g.HasCommitsAhead(w.Path, "main")
	if err != nil {
		return "", fmt.Errorf("check commits ahead: %w", err)
	}
	if !ahead {
		return "", nil
	}

	// Create stash branch at HEAD.
	stashBranch := fmt.Sprintf("stash/%s/%s", w.TrackID, w.Name)
	if err := g.CreateStashBranch(w.Path, stashBranch); err != nil {
		return "", fmt.Errorf("create stash branch %s: %w", stashBranch, err)
	}

	return stashBranch, nil
}

// Return resets a worktree to idle state, stashing work and cleaning up the implementation branch.
func (p *Pool) Return(w *Worktree) error {
	g := p.git()

	trackID := w.TrackID

	// Stash work before cleanup if there's an active track.
	if trackID != "" {
		if _, err := p.Stash(w); err != nil {
			// Log but don't fail return — stash is best-effort.
			_ = err
		}
	}

	// Checkout the pool branch.
	if err := g.CheckoutBranch(w.Path, w.Branch); err != nil {
		return fmt.Errorf("checkout pool branch %s: %w", w.Branch, err)
	}

	// Reset to main.
	if err := g.ResetHardMain(w.Path); err != nil {
		return fmt.Errorf("reset to main: %w", err)
	}

	// Delete implementation branch if it existed.
	if trackID != "" {
		_ = g.DeleteBranch(trackID) // best effort
	}

	w.Status = StatusIdle
	w.TrackID = ""
	w.AgentID = ""
	w.AcquiredAt = nil
	return nil
}

// CleanupStash deletes all stash branches for the given track.
func (p *Pool) CleanupStash(trackID string) error {
	g := p.git()
	stashes, err := g.ListStashBranches(trackID)
	if err != nil {
		return fmt.Errorf("list stash branches for %s: %w", trackID, err)
	}
	if len(stashes) == 0 {
		return nil
	}
	return g.DeleteBranches(stashes)
}

// StashByTrackID finds the worktree for the given track and stashes its work.
func (p *Pool) StashByTrackID(trackID string) error {
	w := p.FindByTrackID(trackID)
	if w == nil {
		return fmt.Errorf("no worktree found for track %s", trackID)
	}
	_, err := p.Stash(w)
	return err
}

// FindByTrackID returns the worktree assigned to the given track.
func (p *Pool) FindByTrackID(trackID string) *Worktree {
	for _, w := range p.Worktrees {
		if w.TrackID == trackID {
			return w
		}
	}
	return nil
}

// ReturnByTrackID finds and returns the worktree assigned to the given track.
func (p *Pool) ReturnByTrackID(trackID string) error {
	w := p.FindByTrackID(trackID)
	if w == nil {
		return fmt.Errorf("no worktree found for track %s", trackID)
	}
	return p.Return(w)
}

// Status returns a sorted list of all worktree states.
func (p *Pool) Status() []Worktree {
	result := make([]Worktree, 0, len(p.Worktrees))
	for _, name := range sortedKeys(p.Worktrees) {
		result = append(result, *p.Worktrees[name])
	}
	return result
}

// Save writes pool state to the data directory.
func (p *Pool) Save(dataDir string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pool state: %w", err)
	}
	path := filepath.Join(dataDir, poolFileName)
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// EnsureSkillsSymlink creates a symlink at {worktreeDir}/.claude/skills pointing
// to {projectDir}/.claude/skills/ so that the Claude SDK can discover skills
// installed in the project directory when running from a worktree.
// Returns nil if the symlink already exists and points to the correct target,
// if the project has no skills directory, or if the worktree is the project dir.
func EnsureSkillsSymlink(worktreeDir, projectDir string) error {
	if worktreeDir == projectDir {
		return nil
	}

	srcSkills := filepath.Join(projectDir, ".claude", "skills")
	if _, err := os.Stat(srcSkills); os.IsNotExist(err) {
		return nil // no skills to link
	}

	destClaudeDir := filepath.Join(worktreeDir, ".claude")
	destSkills := filepath.Join(destClaudeDir, "skills")

	// Check if symlink already exists and points to the right place.
	if target, err := os.Readlink(destSkills); err == nil {
		if target == srcSkills {
			return nil // already correct
		}
		// Points to wrong target — remove and recreate.
		os.Remove(destSkills)
	} else if fi, err := os.Stat(destSkills); err == nil {
		if fi.IsDir() {
			// Real directory exists — don't overwrite, skills may have been
			// installed directly into the worktree.
			return nil
		}
		os.Remove(destSkills)
	}

	if err := os.MkdirAll(destClaudeDir, 0o755); err != nil {
		return fmt.Errorf("create .claude dir in worktree: %w", err)
	}

	return os.Symlink(srcSkills, destSkills)
}

func sortedKeys(m map[string]*Worktree) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
