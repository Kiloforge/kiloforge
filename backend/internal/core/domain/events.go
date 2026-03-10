package domain

// Event type constants for the event bus.
const (
	EventAgentUpdate           = "agent_update"
	EventAgentRemoved          = "agent_removed"
	EventQuotaUpdate           = "quota_update"
	EventTrackUpdate           = "track_update"
	EventTrackRemoved          = "track_removed"
	EventBoardUpdate           = "board_update"
	EventTraceUpdate           = "trace_update"
	EventProjectUpdate         = "project_update"
	EventProjectRemoved        = "project_removed"
	EventLockUpdate            = "lock_update"
	EventLockReleased          = "lock_released"
	EventQueueUpdate           = "queue_update"
	EventCapacityChanged       = "capacity_changed"
	EventProjectSettingsUpdate = "project_settings_update"
	EventReliabilityEvent      = "reliability_event"
	EventNotificationCreated   = "notification_created"
	EventNotificationDismissed = "notification_dismissed"
)

// NewAgentUpdateEvent creates an agent_update event.
func NewAgentUpdateEvent(data any) Event {
	return Event{Type: EventAgentUpdate, Data: data}
}

// NewAgentRemovedEvent creates an agent_removed event.
func NewAgentRemovedEvent(id string) Event {
	return Event{Type: EventAgentRemoved, Data: map[string]string{"id": id}}
}

// NewQuotaUpdateEvent creates a quota_update event.
func NewQuotaUpdateEvent(data any) Event {
	return Event{Type: EventQuotaUpdate, Data: data}
}

// NewTrackUpdateEvent creates a track_update event.
func NewTrackUpdateEvent(data any) Event {
	return Event{Type: EventTrackUpdate, Data: data}
}

// NewTrackRemovedEvent creates a track_removed event.
func NewTrackRemovedEvent(id string) Event {
	return Event{Type: EventTrackRemoved, Data: map[string]string{"id": id}}
}

// NewBoardUpdateEvent creates a board_update event.
func NewBoardUpdateEvent(data any) Event {
	return Event{Type: EventBoardUpdate, Data: data}
}

// NewTraceUpdateEvent creates a trace_update event.
func NewTraceUpdateEvent(data any) Event {
	return Event{Type: EventTraceUpdate, Data: data}
}

// NewProjectUpdateEvent creates a project_update event.
func NewProjectUpdateEvent(data any) Event {
	return Event{Type: EventProjectUpdate, Data: data}
}

// NewProjectRemovedEvent creates a project_removed event.
func NewProjectRemovedEvent(slug string) Event {
	return Event{Type: EventProjectRemoved, Data: map[string]string{"slug": slug}}
}

// NewLockUpdateEvent creates a lock_update event.
func NewLockUpdateEvent(data any) Event {
	return Event{Type: EventLockUpdate, Data: data}
}

// NewLockReleasedEvent creates a lock_released event.
func NewLockReleasedEvent(scope string) Event {
	return Event{Type: EventLockReleased, Data: map[string]string{"scope": scope}}
}

// NewQueueUpdateEvent creates a queue_update event.
func NewQueueUpdateEvent(action string, data any) Event {
	return Event{Type: EventQueueUpdate, Data: map[string]any{"action": action, "data": data}}
}

// NewCapacityChangedEvent creates a capacity_changed event.
func NewCapacityChangedEvent(capacity SwarmCapacity) Event {
	return Event{Type: EventCapacityChanged, Data: capacity}
}

// NewProjectSettingsUpdateEvent creates a project_settings_update event.
func NewProjectSettingsUpdateEvent(slug string, data any) Event {
	return Event{Type: EventProjectSettingsUpdate, Data: map[string]any{"slug": slug, "settings": data}}
}

// NewReliabilityEventEvent creates a reliability_event event for SSE streaming.
func NewReliabilityEventEvent(data any) Event {
	return Event{Type: EventReliabilityEvent, Data: data}
}

// NewNotificationCreatedEvent creates a notification_created event.
func NewNotificationCreatedEvent(n Notification) Event {
	return Event{Type: EventNotificationCreated, Data: map[string]any{
		"id":       n.ID,
		"agent_id": n.AgentID,
		"title":    n.Title,
		"body":     n.Body,
	}}
}

// NewNotificationDismissedEvent creates a notification_dismissed event.
func NewNotificationDismissedEvent(agentID string) Event {
	return Event{Type: EventNotificationDismissed, Data: map[string]string{
		"agent_id": agentID,
	}}
}
