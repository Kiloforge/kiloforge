package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	defaultEndpoint = "https://us.i.posthog.com/capture"
	bufferSize      = 256
	sendTimeout     = 5 * time.Second
	drainTimeout    = 3 * time.Second
)

type captureEvent struct {
	APIKey     string         `json:"api_key"`
	Event      string         `json:"event"`
	DistinctID string         `json:"distinct_id"`
	Properties map[string]any `json:"properties,omitempty"`
	Timestamp  string         `json:"timestamp"`
}

// PostHogTracker sends analytics events to PostHog via HTTP.
// Events are queued in a buffered channel and sent by a background goroutine.
type PostHogTracker struct {
	apiKey     string
	distinctID string
	endpoint   string
	client     *http.Client
	events     chan captureEvent
	wg         sync.WaitGroup
	done       chan struct{}
}

// NewPostHogTracker creates a tracker that sends events to PostHog.
// Call Shutdown to drain buffered events on exit.
func NewPostHogTracker(apiKey, distinctID string) *PostHogTracker {
	return newPostHogTracker(apiKey, distinctID, defaultEndpoint, &http.Client{Timeout: sendTimeout})
}

func newPostHogTracker(apiKey, distinctID, endpoint string, client *http.Client) *PostHogTracker {
	t := &PostHogTracker{
		apiKey:     apiKey,
		distinctID: distinctID,
		endpoint:   endpoint,
		client:     client,
		events:     make(chan captureEvent, bufferSize),
		done:       make(chan struct{}),
	}
	t.wg.Add(1)
	go t.sender()
	return t
}

// Track queues an event for async delivery. Non-blocking: drops events if buffer is full.
func (t *PostHogTracker) Track(_ context.Context, event string, props map[string]any) {
	ev := captureEvent{
		APIKey:     t.apiKey,
		Event:      event,
		DistinctID: t.distinctID,
		Properties: props,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	select {
	case t.events <- ev:
	default:
		// Buffer full — drop event silently.
	}
}

// Shutdown drains remaining events and stops the sender goroutine.
func (t *PostHogTracker) Shutdown(_ context.Context) error {
	close(t.done)
	t.wg.Wait()
	return nil
}

func (t *PostHogTracker) sender() {
	defer t.wg.Done()
	for {
		select {
		case ev := <-t.events:
			t.send(ev)
		case <-t.done:
			// Drain remaining events with a timeout.
			timer := time.NewTimer(drainTimeout)
			defer timer.Stop()
			for {
				select {
				case ev := <-t.events:
					t.send(ev)
				case <-timer.C:
					return
				default:
					return
				}
			}
		}
	}
}

func (t *PostHogTracker) send(ev captureEvent) {
	body, err := json.Marshal(ev)
	if err != nil {
		return
	}
	resp, err := t.client.Post(t.endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[analytics] send failed: %v", err)
		return
	}
	resp.Body.Close()
}
