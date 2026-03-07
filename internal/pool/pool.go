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

	w.TrackID = trackID
	return nil
}

// Return resets a worktree to idle state, cleaning up the implementation branch.
func (p *Pool) Return(w *Worktree) error {
	g := p.git()

	trackID := w.TrackID

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

func sortedKeys(m map[string]*Worktree) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
