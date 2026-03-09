package domain

import "time"

// QueueItem represents a track in the work queue.
type QueueItem struct {
	TrackID     string     `json:"track_id"`
	ProjectSlug string    `json:"project_slug"`
	Status      string     `json:"status"` // queued, assigned, completed, failed
	AgentID     string     `json:"agent_id,omitempty"`
	EnqueuedAt  time.Time  `json:"enqueued_at"`
	AssignedAt  *time.Time `json:"assigned_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Queue status constants.
const (
	QueueStatusQueued    = "queued"
	QueueStatusAssigned  = "assigned"
	QueueStatusCompleted = "completed"
	QueueStatusFailed    = "failed"
)
