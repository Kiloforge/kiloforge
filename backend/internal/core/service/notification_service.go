package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"kiloforge/internal/core/domain"
	"kiloforge/internal/core/port"
)

// NotificationService manages agent-needs-attention notifications.
type NotificationService struct {
	store    port.NotificationStore
	eventBus port.EventBus
}

// NewNotificationService creates a new notification service.
func NewNotificationService(store port.NotificationStore, eventBus port.EventBus) *NotificationService {
	return &NotificationService{store: store, eventBus: eventBus}
}

// Create creates a notification for an agent. Deduplicates: if an active notification
// already exists for this agent, no new notification is created.
func (s *NotificationService) Create(agentID, title, body string) error {
	existing, err := s.store.FindActiveByAgent(agentID)
	if err != nil {
		return fmt.Errorf("check existing notification: %w", err)
	}
	if existing != nil {
		return nil // already has active notification
	}

	n := domain.Notification{
		ID:        uuid.New().String(),
		AgentID:   agentID,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.Insert(n); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(domain.NewNotificationCreatedEvent(n))
	}
	return nil
}

// DismissForAgent dismisses the active notification for an agent (e.g., when they resume work).
// No-op if no active notification exists.
func (s *NotificationService) DismissForAgent(agentID string) error {
	existing, err := s.store.FindActiveByAgent(agentID)
	if err != nil {
		return fmt.Errorf("find notification: %w", err)
	}
	if existing == nil {
		return nil
	}

	if err := s.store.DeleteForAgent(agentID); err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}

	if s.eventBus != nil {
		s.eventBus.Publish(domain.NewNotificationDismissedEvent(agentID))
	}
	return nil
}

// Acknowledge marks a notification as acknowledged by the user.
func (s *NotificationService) Acknowledge(id string) error {
	return s.store.Acknowledge(id)
}

// CleanForAgent removes all notifications for an agent and publishes a dismissed event.
// Used when an agent enters terminal status.
func (s *NotificationService) CleanForAgent(agentID string) error {
	existing, err := s.store.FindActiveByAgent(agentID)
	if err != nil {
		return fmt.Errorf("find notification: %w", err)
	}

	if err := s.store.DeleteForAgent(agentID); err != nil {
		return fmt.Errorf("clean notifications: %w", err)
	}

	if existing != nil && s.eventBus != nil {
		s.eventBus.Publish(domain.NewNotificationDismissedEvent(agentID))
	}
	return nil
}

// ListActive returns all active notifications, optionally filtered by agent ID.
func (s *NotificationService) ListActive(agentID string) ([]domain.Notification, error) {
	return s.store.ListActive(agentID)
}
