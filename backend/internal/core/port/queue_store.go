package port

import "kiloforge/internal/core/domain"

// QueueStore manages work queue persistence.
type QueueStore interface {
	// Enqueue adds a track to the queue with status "queued".
	Enqueue(item domain.QueueItem) error
	// Dequeue removes a track from the queue.
	Dequeue(trackID string) error
	// Assign marks a queued track as assigned to an agent.
	Assign(trackID, agentID string) error
	// Complete marks an assigned track as completed.
	Complete(trackID string) error
	// Fail marks an assigned track as failed.
	Fail(trackID string) error
	// List returns all queue items, optionally filtered by status.
	List(statuses ...string) ([]domain.QueueItem, error)
	// ListPaginated returns a paginated list of queue items, optionally filtered by project and statuses.
	ListPaginated(opts domain.PageOpts, projectSlug string, statuses ...string) (domain.Page[domain.QueueItem], error)
	// Get returns a single queue item by track ID.
	Get(trackID string) (*domain.QueueItem, error)
	// Clear removes all items from the queue.
	Clear() error
}
