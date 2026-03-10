package service

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// --- Mock implementations ---

type mockQueueStore struct {
	mu    sync.Mutex
	items map[string]domain.QueueItem
}

func newMockQueueStore() *mockQueueStore {
	return &mockQueueStore{items: make(map[string]domain.QueueItem)}
}

func (m *mockQueueStore) Enqueue(item domain.QueueItem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[item.TrackID]; ok {
		return nil // INSERT OR IGNORE
	}
	m.items[item.TrackID] = item
	return nil
}

func (m *mockQueueStore) Dequeue(trackID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[trackID]; !ok {
		return fmt.Errorf("not found: %s", trackID)
	}
	delete(m.items, trackID)
	return nil
}

func (m *mockQueueStore) Assign(trackID, agentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[trackID]
	if !ok || item.Status != domain.QueueStatusQueued {
		return fmt.Errorf("not found or not queued: %s", trackID)
	}
	now := time.Now()
	item.Status = domain.QueueStatusAssigned
	item.AgentID = agentID
	item.AssignedAt = &now
	m.items[trackID] = item
	return nil
}

func (m *mockQueueStore) Complete(trackID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[trackID]
	if !ok || item.Status != domain.QueueStatusAssigned {
		return fmt.Errorf("not found or not assigned: %s", trackID)
	}
	now := time.Now()
	item.Status = domain.QueueStatusCompleted
	item.CompletedAt = &now
	m.items[trackID] = item
	return nil
}

func (m *mockQueueStore) Fail(trackID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[trackID]
	if !ok {
		return fmt.Errorf("not found: %s", trackID)
	}
	now := time.Now()
	item.Status = domain.QueueStatusFailed
	item.CompletedAt = &now
	m.items[trackID] = item
	return nil
}

func (m *mockQueueStore) List(statuses ...string) ([]domain.QueueItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	statusSet := make(map[string]bool)
	for _, s := range statuses {
		statusSet[s] = true
	}
	var result []domain.QueueItem
	for _, item := range m.items {
		if len(statuses) == 0 || statusSet[item.Status] {
			result = append(result, item)
		}
	}
	return result, nil
}

func (m *mockQueueStore) Get(trackID string) (*domain.QueueItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[trackID]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (m *mockQueueStore) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]domain.QueueItem)
	return nil
}

func (m *mockQueueStore) ListPaginated(opts domain.PageOpts, _ string, _ ...string) (domain.Page[domain.QueueItem], error) {
	items, err := m.List()
	if err != nil {
		return domain.Page[domain.QueueItem]{}, err
	}
	return domain.Page[domain.QueueItem]{Items: items, TotalCount: len(items)}, nil
}

type mockTrackReader struct {
	tracks []port.TrackEntry
}

func (m *mockTrackReader) DiscoverTracks(_ string) ([]port.TrackEntry, error) {
	return m.tracks, nil
}
func (m *mockTrackReader) DiscoverTracksPaginated(_ string, _ domain.PageOpts, _ ...string) (domain.Page[port.TrackEntry], error) {
	return domain.Page[port.TrackEntry]{Items: m.tracks, TotalCount: len(m.tracks)}, nil
}
func (m *mockTrackReader) GetTrackDetail(_, _ string) (*port.TrackDetail, error) { return nil, nil }
func (m *mockTrackReader) RemoveTrack(_, _ string) error                         { return nil }
func (m *mockTrackReader) IsInitialized(_ string) bool                           { return true }

type mockSpawner struct {
	mu      sync.Mutex
	spawned []string
	nextID  int
	failOn  map[string]bool
}

func newMockSpawner() *mockSpawner {
	return &mockSpawner{failOn: make(map[string]bool)}
}

func (m *mockSpawner) SpawnDeveloper(_ context.Context, opts SpawnDeveloperOpts) (*domain.AgentInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failOn[opts.TrackID] {
		return nil, fmt.Errorf("spawn failed for %s", opts.TrackID)
	}
	m.nextID++
	id := fmt.Sprintf("agent-%d", m.nextID)
	m.spawned = append(m.spawned, opts.TrackID)
	return &domain.AgentInfo{
		ID:  id,
		Ref: opts.TrackID,
	}, nil
}

func (m *mockSpawner) getSpawned() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]string, len(m.spawned))
	copy(cp, m.spawned)
	return cp
}

type mockPool struct {
	mu           sync.Mutex
	available    int
	returned     []string
	cleanedStash []string
}

func newMockPool(n int) *mockPool {
	return &mockPool{available: n}
}

func (m *mockPool) Acquire() (*ImplementWorktree, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.available <= 0 {
		return nil, fmt.Errorf("no worktrees available")
	}
	m.available--
	return &ImplementWorktree{Path: "/tmp/wt"}, nil
}

func (m *mockPool) Prepare(_ *ImplementWorktree, _ string) error { return nil }

func (m *mockPool) ReturnByTrackID(trackID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available++
	m.returned = append(m.returned, trackID)
	return nil
}

func (m *mockPool) CleanupStash(trackID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanedStash = append(m.cleanedStash, trackID)
	return nil
}

func (m *mockPool) Save(_ string) error { return nil }

type mockEventBus struct {
	mu     sync.Mutex
	events []domain.Event
}

func (m *mockEventBus) Publish(e domain.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, e)
}

func (m *mockEventBus) Subscribe() <-chan domain.Event    { return make(chan domain.Event) }
func (m *mockEventBus) Unsubscribe(_ <-chan domain.Event) {}
func (m *mockEventBus) ClientCount() int                  { return 0 }

func (m *mockEventBus) getEvents() []domain.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]domain.Event, len(m.events))
	copy(cp, m.events)
	return cp
}

// --- Tests ---

func TestQueueService_StartStop(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  2,
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       store,
		TrackReader: &mockTrackReader{},
		Spawner:     newMockSpawner(),
		Pool:        newMockPool(3),
	})

	if svc.IsRunning() {
		t.Fatal("should not be running initially")
	}

	if err := svc.Start(context.Background(), "test-project"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !svc.IsRunning() {
		t.Fatal("should be running after start")
	}

	// Starting again should error.
	if err := svc.Start(context.Background(), ""); err == nil {
		t.Fatal("expected error on double start")
	}

	if err := svc.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if svc.IsRunning() {
		t.Fatal("should not be running after stop")
	}

	// Stopping again should error.
	if err := svc.Stop(); err == nil {
		t.Fatal("expected error on double stop")
	}
}

func TestQueueService_EnqueueAndNext(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  2,
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       store,
		TrackReader: &mockTrackReader{},
		Spawner:     newMockSpawner(),
		Pool:        newMockPool(3),
	})

	// Enqueue two tracks.
	if err := svc.Enqueue("track-a", "proj"); err != nil {
		t.Fatalf("enqueue a: %v", err)
	}
	if err := svc.Enqueue("track-b", "proj"); err != nil {
		t.Fatalf("enqueue b: %v", err)
	}

	// Next should return one.
	item, err := svc.Next()
	if err != nil {
		t.Fatalf("next: %v", err)
	}
	if item == nil {
		t.Fatal("expected item, got nil")
	}
}

func TestQueueService_SemaphoreEnforcement(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	spawner := newMockSpawner()
	pool := newMockPool(5)
	trackReader := &mockTrackReader{
		tracks: []port.TrackEntry{
			{ID: "t1", Status: StatusPending},
			{ID: "t2", Status: StatusPending},
			{ID: "t3", Status: StatusPending},
		},
	}

	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  2, // Only allow 2 concurrent
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       store,
		TrackReader: trackReader,
		Spawner:     spawner,
		Pool:        pool,
	})

	if err := svc.Start(context.Background(), "proj"); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Should have spawned exactly 2 (semaphore limit).
	spawned := spawner.getSpawned()
	if len(spawned) != 2 {
		t.Fatalf("expected 2 spawned, got %d: %v", len(spawned), spawned)
	}
}

func TestQueueService_OnAgentComplete_SpawnsNext(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	spawner := newMockSpawner()
	pool := newMockPool(5)
	bus := &mockEventBus{}
	trackReader := &mockTrackReader{
		tracks: []port.TrackEntry{
			{ID: "t1", Status: StatusPending},
			{ID: "t2", Status: StatusPending},
			{ID: "t3", Status: StatusPending},
		},
	}

	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  1, // Only 1 at a time
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       store,
		TrackReader: trackReader,
		EventBus:    bus,
		Spawner:     spawner,
		Pool:        pool,
	})

	if err := svc.Start(context.Background(), "proj"); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Should have spawned 1 initially.
	if len(spawner.getSpawned()) != 1 {
		t.Fatalf("expected 1 spawned initially, got %d", len(spawner.getSpawned()))
	}

	// Complete the first track — should auto-spawn next.
	firstTrack := spawner.getSpawned()[0]
	svc.OnAgentComplete(context.Background(), firstTrack, "completed")

	// Should now have spawned 2 total.
	if len(spawner.getSpawned()) != 2 {
		t.Fatalf("expected 2 spawned after completion, got %d", len(spawner.getSpawned()))
	}

	// Verify events were published.
	events := bus.getEvents()
	if len(events) == 0 {
		t.Fatal("expected events to be published")
	}
}

func TestQueueService_Status(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  3,
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       store,
		TrackReader: &mockTrackReader{},
		Spawner:     newMockSpawner(),
		Pool:        newMockPool(3),
	})

	status, err := svc.Status()
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.Running {
		t.Error("should not be running")
	}
	if status.MaxWorkers != 3 {
		t.Errorf("max_workers = %d, want 3", status.MaxWorkers)
	}
	if status.ActiveWorkers != 0 {
		t.Errorf("active_workers = %d, want 0", status.ActiveWorkers)
	}
}

func TestQueueService_SetMaxWorkers(t *testing.T) {
	t.Parallel()

	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  3,
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       newMockQueueStore(),
		TrackReader: &mockTrackReader{},
		Spawner:     newMockSpawner(),
		Pool:        newMockPool(3),
	})

	if svc.MaxWorkers() != 3 {
		t.Fatalf("initial max_workers = %d, want 3", svc.MaxWorkers())
	}

	svc.SetMaxWorkers(5)
	if svc.MaxWorkers() != 5 {
		t.Fatalf("updated max_workers = %d, want 5", svc.MaxWorkers())
	}
}

func TestQueueService_OnAgentComplete_CleansStash(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	spawner := newMockSpawner()
	pool := newMockPool(5)
	trackReader := &mockTrackReader{
		tracks: []port.TrackEntry{
			{ID: "t1", Status: StatusPending},
		},
	}

	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  1,
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       store,
		TrackReader: trackReader,
		Spawner:     spawner,
		Pool:        pool,
	})

	if err := svc.Start(context.Background(), "proj"); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Complete the track.
	svc.OnAgentComplete(context.Background(), "t1", "completed")

	// Verify stash cleanup was called.
	pool.mu.Lock()
	cleaned := make([]string, len(pool.cleanedStash))
	copy(cleaned, pool.cleanedStash)
	pool.mu.Unlock()

	if len(cleaned) != 1 || cleaned[0] != "t1" {
		t.Errorf("cleanedStash = %v, want [t1]", cleaned)
	}
}

func TestQueueService_OnAgentFail_NoCleanStash(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	spawner := newMockSpawner()
	pool := newMockPool(5)
	trackReader := &mockTrackReader{
		tracks: []port.TrackEntry{
			{ID: "t1", Status: StatusPending},
		},
	}

	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers:  1,
		ProjectDir:  "/tmp/test",
		DataDir:     "/tmp/data",
		Store:       store,
		TrackReader: trackReader,
		Spawner:     spawner,
		Pool:        pool,
	})

	if err := svc.Start(context.Background(), "proj"); err != nil {
		t.Fatalf("start: %v", err)
	}

	// Fail the track (not complete).
	svc.OnAgentComplete(context.Background(), "t1", "failed")

	// Verify stash cleanup was NOT called on failure.
	pool.mu.Lock()
	cleaned := make([]string, len(pool.cleanedStash))
	copy(cleaned, pool.cleanedStash)
	pool.mu.Unlock()

	if len(cleaned) != 0 {
		t.Errorf("cleanedStash should be empty on failure, got %v", cleaned)
	}
}

func TestQueueService_SpawnFailure_MarksFailed(t *testing.T) {
	t.Parallel()

	store := newMockQueueStore()
	spawner := newMockSpawner()
	spawner.failOn["t1"] = true

	svc := NewQueueService(QueueServiceOpts{
		MaxWorkers: 2,
		ProjectDir: "/tmp/test",
		DataDir:    "/tmp/data",
		Store:      store,
		TrackReader: &mockTrackReader{
			tracks: []port.TrackEntry{
				{ID: "t1", Status: StatusPending},
				{ID: "t2", Status: StatusPending},
			},
		},
		Spawner: spawner,
		Pool:    newMockPool(3),
	})

	if err := svc.Start(context.Background(), "proj"); err != nil {
		t.Fatalf("start: %v", err)
	}

	// t1 should be failed, t2 should be spawned.
	item, _ := store.Get("t1")
	if item != nil && item.Status != domain.QueueStatusFailed {
		t.Errorf("t1 status = %q, want %q", item.Status, domain.QueueStatusFailed)
	}

	spawned := spawner.getSpawned()
	found := false
	for _, s := range spawned {
		if s == "t2" {
			found = true
		}
	}
	if !found {
		t.Error("expected t2 to be spawned after t1 failure")
	}
}
