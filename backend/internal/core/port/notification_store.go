package port

import "kiloforge/internal/core/domain"

// NotificationStore persists and queries agent attention notifications.
type NotificationStore interface {
	// Insert records a new notification.
	Insert(n domain.Notification) error
	// ListActive returns unacknowledged notifications, optionally filtered by agent ID.
	ListActive(agentID string) ([]domain.Notification, error)
	// Acknowledge marks a notification as acknowledged.
	Acknowledge(id string) error
	// DeleteForAgent removes all notifications for an agent.
	DeleteForAgent(agentID string) error
	// FindActiveByAgent returns the active notification for an agent, or nil.
	FindActiveByAgent(agentID string) (*domain.Notification, error)
}
