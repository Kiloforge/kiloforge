package domain

import "time"

// Notification represents an agent-needs-attention notification.
type Notification struct {
	ID             string     `json:"id"`
	AgentID        string     `json:"agent_id"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"created_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}
