package lock

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrTimeout   = errors.New("timeout waiting for lock")
	ErrNotHolder = errors.New("lock not held by this holder")
	ErrNotFound  = errors.New("lock not found")
)

// Lock represents a held scoped lock.
type Lock struct {
	Scope      string    `json:"scope"`
	Holder     string    `json:"holder"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Manager provides thread-safe scoped lock management with TTL and heartbeat.
type Manager struct {
	mu      sync.Mutex
	locks   map[string]*Lock
	waiters map[string][]chan struct{}
	dataDir string
}

const lockFile = "locks.json"

// New creates a lock manager. If dataDir is non-empty, persisted locks are
// loaded (expired ones discarded).
func New(dataDir string) *Manager {
	m := &Manager{
		locks:   make(map[string]*Lock),
		waiters: make(map[string][]chan struct{}),
		dataDir: dataDir,
	}
	if dataDir != "" {
		_ = m.load()
	}
	return m
}

// Acquire attempts to acquire a named lock. If the lock is held by another
// holder, it blocks until the lock is freed or ctx is cancelled.
func (m *Manager) Acquire(ctx context.Context, scope, holder string, ttl time.Duration) (*Lock, error) {
	m.mu.Lock()

	// Check if lock is free or expired.
	if existing, ok := m.locks[scope]; !ok || time.Now().After(existing.ExpiresAt) {
		now := time.Now()
		l := &Lock{
			Scope:      scope,
			Holder:     holder,
			AcquiredAt: now,
			ExpiresAt:  now.Add(ttl),
		}
		m.locks[scope] = l
		m.mu.Unlock()
		_ = m.save()
		return l, nil
	}

	// Already held by this holder — re-entrant acquire: extend TTL.
	if m.locks[scope].Holder == holder {
		m.locks[scope].ExpiresAt = time.Now().Add(ttl)
		l := *m.locks[scope]
		m.mu.Unlock()
		_ = m.save()
		return &l, nil
	}

	// Lock held — register waiter.
	ch := make(chan struct{}, 1)
	m.waiters[scope] = append(m.waiters[scope], ch)
	m.mu.Unlock()

	select {
	case <-ch:
		return m.Acquire(ctx, scope, holder, ttl)
	case <-ctx.Done():
		// Remove our channel from waiters.
		m.mu.Lock()
		ws := m.waiters[scope]
		for i, w := range ws {
			if w == ch {
				m.waiters[scope] = append(ws[:i], ws[i+1:]...)
				break
			}
		}
		m.mu.Unlock()
		return nil, ErrTimeout
	}
}

// Release releases a lock. Only the holder can release it.
func (m *Manager) Release(scope, holder string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.locks[scope]
	if !ok {
		return ErrNotFound
	}
	if existing.Holder != holder {
		return ErrNotHolder
	}

	delete(m.locks, scope)
	m.notifyWaiters(scope)
	go func() { _ = m.save() }()
	return nil
}

// Heartbeat extends the TTL of a held lock.
func (m *Manager) Heartbeat(scope, holder string, ttl time.Duration) (*Lock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.locks[scope]
	if !ok || time.Now().After(existing.ExpiresAt) {
		return nil, ErrNotFound
	}
	if existing.Holder != holder {
		return nil, ErrNotHolder
	}

	existing.ExpiresAt = time.Now().Add(ttl)
	l := *existing
	return &l, nil
}

// List returns all active (non-expired) locks.
func (m *Manager) List() []Lock {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var result []Lock
	for _, l := range m.locks {
		if now.Before(l.ExpiresAt) {
			result = append(result, *l)
		}
	}
	return result
}

// StartReaper runs a background goroutine that expires stale locks.
func (m *Manager) StartReaper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.reap()
			}
		}
	}()
}

func (m *Manager) reap() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for scope, l := range m.locks {
		if now.After(l.ExpiresAt) {
			delete(m.locks, scope)
			m.notifyWaiters(scope)
		}
	}
}

func (m *Manager) notifyWaiters(scope string) {
	ws := m.waiters[scope]
	if len(ws) == 0 {
		return
	}
	// Notify the first waiter.
	ws[0] <- struct{}{}
	m.waiters[scope] = ws[1:]
}

// Persistence

type lockState struct {
	Locks []*Lock `json:"locks"`
}

func (m *Manager) save() error {
	if m.dataDir == "" {
		return nil
	}
	m.mu.Lock()
	state := lockState{}
	now := time.Now()
	for _, l := range m.locks {
		if now.Before(l.ExpiresAt) {
			state.Locks = append(state.Locks, l)
		}
	}
	m.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(m.dataDir, lockFile), data, 0o644)
}

func (m *Manager) load() error {
	path := filepath.Join(m.dataDir, lockFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var state lockState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	now := time.Now()
	for _, l := range state.Locks {
		if now.Before(l.ExpiresAt) {
			m.locks[l.Scope] = l
		}
	}
	return nil
}
