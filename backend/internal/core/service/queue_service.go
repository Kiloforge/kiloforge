package service

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
	"kiloforge/pkg/kf"
)

// QueueSpawner spawns developer agents for the queue.
type QueueSpawner interface {
	SpawnDeveloper(ctx context.Context, opts SpawnDeveloperOpts) (*domain.AgentInfo, error)
}

// SwarmCapacityChecker checks global swarm capacity across all agent types.
// When set, the queue service defers to this instead of its own internal semaphore.
type SwarmCapacityChecker interface {
	CanSpawn() bool
}

// QueueWorktreePool manages worktree acquisition for queued tracks.
type QueueWorktreePool interface {
	Acquire() (*ImplementWorktree, error)
	Prepare(wt *ImplementWorktree, trackID string) error
	ReturnByTrackID(trackID string) error
	Save(dataDir string) error
}

// QueueService manages the work queue with semaphore-based concurrency control
// and dependency-aware scheduling.
type QueueService struct {
	mu          sync.Mutex
	running     bool
	maxWorkers  int
	projectSlug string
	projectDir  string
	dataDir     string

	store           port.QueueStore
	trackReader     port.TrackReader
	eventBus        port.EventBus
	spawner         QueueSpawner
	pool            QueueWorktreePool
	implSvc         *ImplementService
	capacityChecker SwarmCapacityChecker
	logger          *log.Logger
}

// QueueServiceOpts configures the QueueService.
type QueueServiceOpts struct {
	MaxWorkers      int
	ProjectSlug     string
	ProjectDir      string
	DataDir         string
	Store           port.QueueStore
	TrackReader     port.TrackReader
	EventBus        port.EventBus
	Spawner         QueueSpawner
	Pool            QueueWorktreePool
	ImplSvc         *ImplementService
	CapacityChecker SwarmCapacityChecker
	Logger          *log.Logger
}

// NewQueueService creates a new QueueService.
func NewQueueService(opts QueueServiceOpts) *QueueService {
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = 3
	}
	if opts.Logger == nil {
		opts.Logger = log.Default()
	}
	return &QueueService{
		maxWorkers:      opts.MaxWorkers,
		projectSlug:     opts.ProjectSlug,
		projectDir:      opts.ProjectDir,
		dataDir:         opts.DataDir,
		store:           opts.Store,
		trackReader:     opts.TrackReader,
		eventBus:        opts.EventBus,
		spawner:         opts.Spawner,
		pool:            opts.Pool,
		implSvc:         opts.ImplSvc,
		capacityChecker: opts.CapacityChecker,
		logger:          opts.Logger,
	}
}

// IsRunning returns whether the queue is actively scheduling.
func (q *QueueService) IsRunning() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.running
}

// SetMaxWorkers updates the maximum concurrent workers.
func (q *QueueService) SetMaxWorkers(n int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.maxWorkers = n
}

// MaxWorkers returns the current max workers setting.
func (q *QueueService) MaxWorkers() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.maxWorkers
}

// Start begins queue processing: enqueues ready tracks and spawns up to limit.
func (q *QueueService) Start(ctx context.Context, projectSlug string) error {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return fmt.Errorf("queue is already running")
	}
	q.running = true
	if projectSlug != "" {
		q.projectSlug = projectSlug
	}
	q.mu.Unlock()

	q.publishEvent("queue_started", map[string]any{
		"project":     q.projectSlug,
		"max_workers": q.maxWorkers,
	})

	// Enqueue all ready tracks.
	if err := q.EnqueueReady(); err != nil {
		q.logger.Printf("[queue] enqueue ready: %v", err)
	}

	// Spawn up to limit.
	q.SpawnUpToLimit(ctx)
	return nil
}

// Stop halts queue processing. Running agents finish normally.
func (q *QueueService) Stop() error {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return fmt.Errorf("queue is not running")
	}
	q.running = false
	q.mu.Unlock()

	q.publishEvent("queue_stopped", map[string]any{
		"project": q.projectSlug,
	})

	return nil
}

// Enqueue adds a single track to the queue.
func (q *QueueService) Enqueue(trackID, projectSlug string) error {
	item := domain.QueueItem{
		TrackID:     trackID,
		ProjectSlug: projectSlug,
		Status:      domain.QueueStatusQueued,
		EnqueuedAt:  time.Now().UTC(),
	}
	return q.store.Enqueue(item)
}

// EnqueueReady discovers all pending tracks and enqueues those with satisfied deps.
func (q *QueueService) EnqueueReady() error {
	tracks, err := q.trackReader.DiscoverTracks(q.projectDir)
	if err != nil {
		return fmt.Errorf("discover tracks: %w", err)
	}

	// Build completed set for dep checking.
	completedIDs := make(map[string]bool)
	for _, t := range tracks {
		if t.Status == StatusComplete {
			completedIDs[t.ID] = true
		}
	}

	// Read deps graph.
	depsPath := filepath.Join(q.projectDir, ".agent", "kf", "tracks", "deps.yaml")
	graph, err := kf.ReadDepsFile(depsPath)
	if err != nil {
		return fmt.Errorf("read deps: %w", err)
	}

	// Enqueue pending tracks with satisfied deps.
	for _, t := range tracks {
		if t.Status != StatusPending {
			continue
		}
		if !graph.AllDepsSatisfied(t.ID, completedIDs) {
			continue
		}
		if err := q.Enqueue(t.ID, q.projectSlug); err != nil {
			q.logger.Printf("[queue] enqueue %s: %v", t.ID, err)
		}
	}

	return nil
}

// Next returns the next ready track from the queue using dependency-aware ordering.
func (q *QueueService) Next() (*domain.QueueItem, error) {
	items, err := q.store.List(domain.QueueStatusQueued)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	// Build completed set from completed queue items and completed tracks.
	completedIDs := make(map[string]bool)
	allItems, _ := q.store.List(domain.QueueStatusCompleted)
	for _, item := range allItems {
		completedIDs[item.TrackID] = true
	}

	// Also check track statuses for deps completed outside the queue.
	tracks, err := q.trackReader.DiscoverTracks(q.projectDir)
	if err == nil {
		for _, t := range tracks {
			if t.Status == StatusComplete {
				completedIDs[t.ID] = true
			}
		}
	}

	// Read deps graph for ordering.
	depsPath := filepath.Join(q.projectDir, ".agent", "kf", "tracks", "deps.yaml")
	graph, _ := kf.ReadDepsFile(depsPath)

	// Filter to candidates with satisfied deps.
	candidates := make(map[string]bool)
	for _, item := range items {
		if graph.AllDepsSatisfied(item.TrackID, completedIDs) {
			candidates[item.TrackID] = true
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Topological sort for ordering.
	sorted, err := graph.TopologicalSort(candidates)
	if err != nil {
		// Fallback to FIFO if cycle detected.
		for _, item := range items {
			if candidates[item.TrackID] {
				return &item, nil
			}
		}
	}

	// Return first in topological order.
	if len(sorted) > 0 {
		for _, item := range items {
			if item.TrackID == sorted[0] {
				return &item, nil
			}
		}
	}

	return nil, nil
}

// ActiveWorkers returns the count of currently assigned (running) queue items.
func (q *QueueService) ActiveWorkers() int {
	items, err := q.store.List(domain.QueueStatusAssigned)
	if err != nil {
		return 0
	}
	return len(items)
}

// availableSlots returns how many more agents can be spawned.
func (q *QueueService) availableSlots() int {
	q.mu.Lock()
	max := q.maxWorkers
	q.mu.Unlock()
	return max - q.ActiveWorkers()
}

// canSpawnMore returns true if the queue can spawn another agent.
// When a global capacity checker is set, it takes precedence.
func (q *QueueService) canSpawnMore() bool {
	if q.capacityChecker != nil {
		return q.capacityChecker.CanSpawn()
	}
	return q.availableSlots() > 0
}

// SpawnUpToLimit spawns developer agents for queued tracks up to the semaphore limit.
func (q *QueueService) SpawnUpToLimit(ctx context.Context) {
	for q.canSpawnMore() {
		item, err := q.Next()
		if err != nil {
			q.logger.Printf("[queue] next: %v", err)
			return
		}
		if item == nil {
			return // no ready tracks
		}

		if err := q.spawnForTrack(ctx, item); err != nil {
			q.logger.Printf("[queue] spawn %s: %v", item.TrackID, err)
			// Mark as failed so we don't retry endlessly.
			if err := q.store.Fail(item.TrackID); err != nil {
				q.logger.Printf("[queue] fail %s: %v", item.TrackID, err)
			}
		}
	}
}

// spawnForTrack acquires a worktree and spawns a developer agent for a queued track.
func (q *QueueService) spawnForTrack(ctx context.Context, item *domain.QueueItem) error {
	wt, err := q.pool.Acquire()
	if err != nil {
		return fmt.Errorf("acquire worktree: %w", err)
	}

	if err := q.pool.Prepare(wt, item.TrackID); err != nil {
		return fmt.Errorf("prepare worktree: %w", err)
	}

	logDir := filepath.Join(q.dataDir, "projects", item.ProjectSlug, "logs")
	info, err := q.spawner.SpawnDeveloper(ctx, SpawnDeveloperOpts{
		TrackID:     item.TrackID,
		Flags:       "--auto-merge",
		WorktreeDir: wt.Path,
		LogDir:      logDir,
	})
	if err != nil {
		// Return worktree on spawn failure.
		_ = q.pool.ReturnByTrackID(item.TrackID)
		return fmt.Errorf("spawn developer: %w", err)
	}

	// Mark as assigned in queue.
	if err := q.store.Assign(item.TrackID, info.ID); err != nil {
		q.logger.Printf("[queue] assign %s: %v", item.TrackID, err)
	}

	// Move board card to in-progress.
	if q.implSvc != nil {
		if _, _, err := q.implSvc.MoveCardToInProgress(item.ProjectSlug, item.TrackID); err != nil {
			q.logger.Printf("[queue] board move %s: %v", item.TrackID, err)
		}
	}

	wt.AgentID = info.ID
	if err := q.pool.Save(q.dataDir); err != nil {
		q.logger.Printf("[queue] save pool: %v", err)
	}

	q.publishEvent("track_assigned", map[string]any{
		"track_id": item.TrackID,
		"agent_id": info.ID,
	})

	agentShort := info.ID
	if len(agentShort) > 8 {
		agentShort = agentShort[:8]
	}
	q.logger.Printf("[queue] spawned agent %s for track %s", agentShort, item.TrackID)
	return nil
}

// OnAgentComplete is called when a developer agent finishes.
// It updates queue state, re-evaluates deps, and spawns next if slots available.
func (q *QueueService) OnAgentComplete(ctx context.Context, trackID, status string) {
	q.mu.Lock()
	running := q.running
	q.mu.Unlock()

	if !running {
		return
	}

	// Update queue item status.
	if status == "completed" {
		if err := q.store.Complete(trackID); err != nil {
			q.logger.Printf("[queue] complete %s: %v", trackID, err)
		}

		// Move board card to done.
		if q.implSvc != nil {
			if err := q.implSvc.MoveCardToDone(q.projectSlug, trackID); err != nil {
				q.logger.Printf("[queue] board done %s: %v", trackID, err)
			}
		}

		q.publishEvent("track_completed", map[string]any{
			"track_id": trackID,
		})

		// Re-evaluate: newly unblocked tracks may now be enqueueable.
		if err := q.EnqueueReady(); err != nil {
			q.logger.Printf("[queue] re-enqueue: %v", err)
		}
	} else {
		if err := q.store.Fail(trackID); err != nil {
			q.logger.Printf("[queue] fail %s: %v", trackID, err)
		}
	}

	// Return worktree.
	if q.pool != nil {
		if err := q.pool.ReturnByTrackID(trackID); err != nil {
			q.logger.Printf("[queue] return worktree %s: %v", trackID, err)
		}
		if err := q.pool.Save(q.dataDir); err != nil {
			q.logger.Printf("[queue] save pool: %v", err)
		}
	}

	q.publishEvent("worker_freed", map[string]any{
		"track_id": trackID,
		"slots":    q.availableSlots(),
	})

	// Spawn next if slots available.
	q.SpawnUpToLimit(ctx)

	// Check if queue is empty.
	items, err := q.store.List(domain.QueueStatusQueued, domain.QueueStatusAssigned)
	if err == nil && len(items) == 0 {
		q.mu.Lock()
		q.running = false
		q.mu.Unlock()
		q.publishEvent("queue_stopped", map[string]any{
			"project": q.projectSlug,
			"reason":  "all tracks processed",
		})
	}
}

// Status returns the current queue status for API responses.
func (q *QueueService) Status() (*QueueStatus, error) {
	q.mu.Lock()
	running := q.running
	max := q.maxWorkers
	q.mu.Unlock()

	items, err := q.store.List()
	if err != nil {
		return nil, err
	}

	active := 0
	for _, item := range items {
		if item.Status == domain.QueueStatusAssigned {
			active++
		}
	}

	return &QueueStatus{
		Running:       running,
		MaxWorkers:    max,
		ActiveWorkers: active,
		Items:         items,
	}, nil
}

// QueueStatus represents the queue state for API responses.
type QueueStatus struct {
	Running       bool
	MaxWorkers    int
	ActiveWorkers int
	Items         []domain.QueueItem
}

func (q *QueueService) publishEvent(action string, data any) {
	if q.eventBus == nil {
		return
	}
	q.eventBus.Publish(domain.NewQueueUpdateEvent(action, data))
}
