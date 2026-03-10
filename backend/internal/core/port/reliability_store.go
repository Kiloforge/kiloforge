package port

import "kiloforge/internal/core/domain"

// ReliabilityStore persists and queries reliability events.
type ReliabilityStore interface {
	// Insert records a new reliability event.
	Insert(event domain.ReliabilityEvent) error
	// List returns a paginated, filtered list of reliability events.
	List(filter domain.ReliabilityFilter, opts domain.PageOpts) (domain.Page[domain.ReliabilityEvent], error)
	// Summary returns aggregated event counts for the given time window and bucket count.
	Summary(filter domain.ReliabilityFilter, buckets int) (domain.ReliabilitySummary, error)
}

// ReliabilityRecorder records reliability events from subsystems.
// Accepts optional detail as a map[string]any.
type ReliabilityRecorder interface {
	RecordEvent(eventType, severity, agentID, scope string, detail map[string]any) error
}
