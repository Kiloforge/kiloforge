package service

import (
	"time"

	"github.com/google/uuid"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// ReliabilityService records and queries reliability events.
type ReliabilityService struct {
	store    port.ReliabilityStore
	eventBus port.EventBus
}

// NewReliabilityService creates a ReliabilityService.
func NewReliabilityService(store port.ReliabilityStore, eventBus port.EventBus) *ReliabilityService {
	return &ReliabilityService{store: store, eventBus: eventBus}
}

// RecordEvent validates and persists a reliability event, then publishes it to the event bus.
func (s *ReliabilityService) RecordEvent(eventType domain.ReliabilityEventType, severity domain.Severity, agentID, scope string, detail map[string]any) error {
	event := domain.ReliabilityEvent{
		ID:        uuid.NewString(),
		EventType: eventType,
		Severity:  severity,
		AgentID:   agentID,
		Scope:     scope,
		Detail:    detail,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.store.Insert(event); err != nil {
		return err
	}

	if s.eventBus != nil {
		s.eventBus.Publish(domain.NewReliabilityEventEvent(event))
	}

	return nil
}

// ListEvents returns paginated, filtered reliability events.
func (s *ReliabilityService) ListEvents(filter domain.ReliabilityFilter, opts domain.PageOpts) (domain.Page[domain.ReliabilityEvent], error) {
	return s.store.List(filter, opts)
}

// GetSummary returns aggregated reliability event counts.
func (s *ReliabilityService) GetSummary(since, until time.Time, bucket string) (domain.ReliabilitySummary, error) {
	return s.store.Summary(since, until, bucket)
}
