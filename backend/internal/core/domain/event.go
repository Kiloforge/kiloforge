package domain

// Event is a typed event for the publish-subscribe event bus.
type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
