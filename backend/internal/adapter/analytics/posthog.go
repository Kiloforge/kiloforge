package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const (
	defaultCaptureURL = "https://us.i.posthog.com/capture/"
	bufferSize        = 256
	sendTimeout       = 5 * time.Second
	drainTimeout      = 3 * time.Second
)

type capturePayload struct {
	APIKey     string         `json:"api_key"`
	Event      string         `json:"event"`
	DistinctID string         `json:"distinct_id"`
	Properties map[string]any `json:"properties,omitempty"`
	Timestamp  string         `json:"timestamp"`
}

// PostHog sends analytics events to PostHog via HTTP.
type PostHog struct {
	apiKey     string
	distinctID string
	captureURL string
	client     *http.Client
	events     chan capturePayload
	done       chan struct{}
}

// NewPostHog creates a PostHog tracker with a background sender goroutine.
func NewPostHog(apiKey, distinctID string) *PostHog {
	p := &PostHog{
		apiKey:     apiKey,
		distinctID: distinctID,
		captureURL: defaultCaptureURL,
		client:     &http.Client{Timeout: sendTimeout},
		events:     make(chan capturePayload, bufferSize),
		done:       make(chan struct{}),
	}
	go p.sender()
	return p
}

// Track queues an analytics event. Non-blocking: drops if buffer full.
func (p *PostHog) Track(_ context.Context, event string, props map[string]any) {
	payload := capturePayload{
		APIKey:     p.apiKey,
		Event:      event,
		DistinctID: p.distinctID,
		Properties: props,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	select {
	case p.events <- payload:
	default:
	}
}

// Shutdown stops the background sender and drains buffered events.
func (p *PostHog) Shutdown(_ context.Context) error {
	close(p.events)
	select {
	case <-p.done:
	case <-time.After(drainTimeout):
		log.Printf("[analytics] shutdown drain timed out")
	}
	return nil
}

func (p *PostHog) sender() {
	defer close(p.done)
	for payload := range p.events {
		p.send(payload)
	}
}

func (p *PostHog) send(payload capturePayload) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, p.captureURL, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
