package port

import "kiloforge/internal/core/domain"

// EventBus is a publish-subscribe event bus for broadcasting real-time events.
type EventBus interface {
	// Publish sends an event to all subscribers. Non-blocking: slow subscribers
	// that can't keep up will miss events.
	Publish(event domain.Event)

	// Subscribe registers a new subscriber and returns a channel for receiving events.
	Subscribe() <-chan domain.Event

	// Unsubscribe removes a subscriber. The channel is closed after this call.
	Unsubscribe(ch <-chan domain.Event)

	// ClientCount returns the number of active subscribers.
	ClientCount() int
}
