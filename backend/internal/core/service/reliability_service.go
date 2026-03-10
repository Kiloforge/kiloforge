package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// ReliabilityService records and queries agent reliability events.
// Implements port.ReliabilityRecorder.
type ReliabilityService struct {
	store    port.ReliabilityStore
	eventBus port.EventBus
}

// NewReliabilityService creates a new reliability service.
func NewReliabilityService(store port.ReliabilityStore, eventBus port.EventBus) *ReliabilityService {
	return &ReliabilityService{store: store, eventBus: eventBus}
}

// RecordEvent validates and persists a reliability event, then publishes it to the event bus.
// Implements port.ReliabilityRecorder.
func (s *ReliabilityService) RecordEvent(eventType, severity, agentID, scope string, detail map[string]any) error {
	ev := domain.ReliabilityEvent{
		ID:        uuid.New().String(),
		EventType: eventType,
		Severity:  severity,
		AgentID:   agentID,
		Scope:     scope,
		Detail:    detail,
		CreatedAt: time.Now().UTC(),
	}

	if !isValidEventType(eventType) {
		return fmt.Errorf("invalid event type: %s", eventType)
	}
	if !isValidSeverity(severity) {
		return fmt.Errorf("invalid severity: %s", severity)
	}

	if err := s.store.Insert(ev); err != nil {
		return fmt.Errorf("insert reliability event: %w", err)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(domain.NewReliabilityEventEvent(ev))
	}

	return nil
}

// ListEvents returns a paginated, filtered list of reliability events.
func (s *ReliabilityService) ListEvents(filter domain.ReliabilityFilter, opts domain.PageOpts) (domain.Page[domain.ReliabilityEvent], error) {
	return s.store.List(filter, opts)
}

// GetSummary returns aggregated event counts for the given filter and bucket count.
func (s *ReliabilityService) GetSummary(filter domain.ReliabilityFilter, buckets int) (domain.ReliabilitySummary, error) {
	return s.store.Summary(filter, buckets)
}

func isValidEventType(t string) bool {
	for _, v := range domain.ValidReliabilityEventTypes() {
		if v == t {
			return true
		}
	}
	return false
}

func isValidSeverity(s string) bool {
	for _, v := range domain.ValidSeverities() {
		if v == s {
			return true
		}
	}
	return false
}
