package port

import "context"

// AnalyticsTracker sends anonymous product analytics events.
type AnalyticsTracker interface {
	// Track queues an analytics event for async delivery.
	// Implementations must be non-blocking (fire-and-forget).
	Track(ctx context.Context, event string, props map[string]any)

	// Shutdown drains buffered events and releases resources.
	Shutdown(ctx context.Context) error
}
