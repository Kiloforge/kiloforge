package port

import (
	"time"

	"kiloforge/internal/core/domain"
)

// ReliabilityStore persists and queries reliability events.
type ReliabilityStore interface {
	Insert(event domain.ReliabilityEvent) error
	List(filter domain.ReliabilityFilter, opts domain.PageOpts) (domain.Page[domain.ReliabilityEvent], error)
	Summary(since, until time.Time, bucket string) (domain.ReliabilitySummary, error)
}
